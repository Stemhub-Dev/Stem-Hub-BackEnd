package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type ProyectoRepository interface {
	ExisteTipoProyectoActivo(codigoTipoProyecto int64) (bool, error)
	ObtenerAmbitoRolActivo(codRol int64) (string, error)
	ObtenerNombreProyecto(codigoProyecto int64) (string, error)
	ExistenGeneros(codigosGeneros []int64) (bool, error)

	Crear(
		codigoIntegrante int64,
		nombre string,
		descripcion *string,
		codigoEstadoProyecto int64,
		codigoTipoProyecto int64,
		codigosGeneros []int64,
		codRol int64,
		ambitoRol string,
		subirLogo func(codigoProyecto int64) (string, error),
	) (int64, error)

	ExisteProyectoActivo(
		codigoProyecto int64,
	) (bool, error)

	ObtenerDetalle(
		codigoProyecto int64,
	) (*model.Proyecto, string, string, error)

	ExisteEstadoProyectoActivo(
		codigoEstadoProyecto int64,
	) (bool, error)

	ListarGenerosProyecto(
		codigoProyecto int64,
	) ([]dto.GeneroProyectoResponse, error)

	// Actualizar solo toca el logo si cambiarLogo es true. Devuelve el logo
	// que quedó guardado y, si cambió, la clave que reemplazó (las dos
	// leídas en la misma transacción).
	Actualizar(
		codigoProyecto int64,
		nombre string,
		descripcion *string,
		cambiarLogo bool,
		logoObjectKey *string,
		codigoEstadoProyecto int64,
		codigoTipoProyecto int64,
		codigosGeneros []int64,
	) (logoVigente *string, logoReemplazado *string, err error)

	PuedeGestionarCanciones(
		codigoIntegrante int64,
		codigoProyecto int64,
	) (bool, error)

	PuedeRealizarEnProyecto(
		codigoIntegrante int64,
		codigoProyecto int64,
		clavePermiso string,
	) (bool, error)

	EsIntegranteActivo(
		codigoIntegrante int64,
		codigoProyecto int64,
	) (bool, error)

	EsPropietarioActivo(
		codigoIntegrante int64,
		codigoProyecto int64,
	) (bool, error)

	ContarPorIntegrante(
		codigoIntegrante int64,
		filtro dto.ListarProyectosFiltro,
	) (int, error)

	ListarPorIntegrante(
		codigoIntegrante int64,
		filtro dto.ListarProyectosFiltro,
		desplazamiento int,
	) ([]dto.ProyectoListadoResponse, error)

	ListarColaboradores(
		codigoProyecto int64,
	) ([]model.ColaboradorProyecto, error)

	ObtenerColaboradorActivo(
		codigoProyecto int64,
		codigoIntegrante int64,
	) (*model.ColaboradorProyecto, error)

	ActualizarRolIntegrante(
		codigoProyecto int64,
		codigoIntegrante int64,
		codRol int64,
	) error

	DarDeBajaIntegrante(
		codigoProyecto int64,
		codigoIntegrante int64,
	) error

	DarDeBaja(
		codigoProyecto int64,
	) error

	ListarMiParticipacion(
		codigoIntegrante int64,
		participacion string,
	) ([]dto.MiParticipacionProyectoResponse, error)
}

// ErrConfirmacionIncierta envuelve un fallo del COMMIT: no se sabe si la
// transacción quedó aplicada, así que quien subió archivos para ella no
// tiene que borrarlos (podrían estar en uso).
var ErrConfirmacionIncierta = errors.New("no se pudo confirmar la transacción")

type proyectoRepository struct {
	db *sql.DB
}

func NewProyectoRepository(db *sql.DB) ProyectoRepository {
	return &proyectoRepository{
		db: db,
	}
}

func (r *proyectoRepository) ExisteTipoProyectoActivo(
	codigoTipoProyecto int64,
) (bool, error) {

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM tipoproyecto
			WHERE codtipoproy = $1
			  AND fechahorabajatipoproy IS NULL
		)
	`

	var existe bool

	err := r.db.QueryRow(
		query,
		codigoTipoProyecto,
	).Scan(&existe)

	return existe, err
}

func (r *proyectoRepository) ObtenerAmbitoRolActivo(
	codRol int64,
) (string, error) {

	query := `
		SELECT ambitorol
		FROM rol
		WHERE codrol = $1
		  AND fechahorabajarol IS NULL
	`

	var ambito string

	err := r.db.QueryRow(
		query,
		codRol,
	).Scan(&ambito)

	return ambito, err
}

func (r *proyectoRepository) ObtenerNombreProyecto(
	codigoProyecto int64,
) (string, error) {

	var nombre string

	err := r.db.QueryRow(`
		SELECT nombreproyecto
		FROM proyecto
		WHERE codigoproyecto = $1
	`,
		codigoProyecto,
	).Scan(&nombre)

	return nombre, err
}

func (r *proyectoRepository) ExistenGeneros(
	codigosGeneros []int64,
) (bool, error) {

	for _, codigoGenero := range codigosGeneros {

		var existe bool

		err := r.db.QueryRow(`
			SELECT EXISTS (
				SELECT 1
				FROM generomusicalproyecto
				WHERE codigogeneroproy = $1
				  AND fechahorabajageneroproy IS NULL
			)
		`, codigoGenero).Scan(&existe)

		if err != nil {
			return false, err
		}

		if !existe {
			return false, nil
		}
	}

	return true, nil
}

func (r *proyectoRepository) Crear(
	codigoIntegrante int64,
	nombre string,
	descripcion *string,
	codigoEstadoProyecto int64,
	codigoTipoProyecto int64,
	codigosGeneros []int64,
	codRol int64,
	ambitoRol string,
	subirLogo func(codigoProyecto int64) (string, error),
) (int64, error) {

	tx, err := r.db.Begin()

	if err != nil {
		return 0, err
	}

	defer tx.Rollback()

	var codigoProyecto int64

	err = tx.QueryRow(`
		INSERT INTO proyecto (
			nombreproyecto,
			descripcionproyecto,
			logoproyecto,
			codestadoproy,
			codtipoproy
		)
		VALUES ($1, $2, NULL, $3, $4)
		RETURNING codigoproyecto
	`,
		nombre,
		descripcion,
		codigoEstadoProyecto,
		codigoTipoProyecto,
	).Scan(&codigoProyecto)

	if err != nil {
		return 0, err
	}

	for _, codigoGenero := range codigosGeneros {

		_, err = tx.Exec(`
			INSERT INTO proyectogeneromusical (
				codigoproyecto,
				codigogeneroproy
			)
			VALUES ($1, $2)
		`,
			codigoProyecto,
			codigoGenero,
		)

		if err != nil {
			return 0, err
		}
	}

	_, err = tx.Exec(`
		INSERT INTO integranteproyecto (
			codintegrante,
			codigoproyecto,
			codrol,
			fechahoraaltaintegranteproy,
			espropietario,
			ambitorol
		)
		VALUES (
			$1,
			$2,
			$3,
			CURRENT_TIMESTAMP,
			TRUE,
			$4
		)
	`,
		codigoIntegrante,
		codigoProyecto,
		codRol,
		ambitoRol,
	)

	if err != nil {
		return 0, err
	}

	// El logo se sube con la transacción abierta porque su clave en el
	// storage lleva el código del proyecto: si la subida falla, el
	// rollback evita que quede un proyecto creado sin la portada elegida.
	if subirLogo != nil {

		logoObjectKey, err := subirLogo(codigoProyecto)

		if err != nil {
			return 0, err
		}

		_, err = tx.Exec(`
			UPDATE proyecto
			SET logoproyecto = $1
			WHERE codigoproyecto = $2
		`,
			logoObjectKey,
			codigoProyecto,
		)

		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("%w: %w", ErrConfirmacionIncierta, err)
	}

	return codigoProyecto, nil
}

func (r *proyectoRepository) ExisteProyectoActivo(
	codigoProyecto int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM proyecto
			WHERE codigoproyecto = $1
			  AND fechahorabajaproyecto IS NULL
		)
	`, codigoProyecto).Scan(&existe)

	return existe, err
}

func (r *proyectoRepository) PuedeGestionarCanciones(
	codigoIntegrante int64,
	codigoProyecto int64,
) (bool, error) {

	var puede bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM integranteproyecto ip
			WHERE ip.codintegrante = $1
			  AND ip.codigoproyecto = $2
			  AND ip.fechahorabajaintegranteproy IS NULL
			  AND EXISTS (
					SELECT 1
					FROM rolpermiso rp
					JOIN permiso p
					  ON p.codigopermiso = rp.codigopermiso
					WHERE rp.codrol = ip.codrol
					  AND rp.ambitorolpermiso = ip.ambitorol
					  AND rp.fechahorabajarolpermiso IS NULL
					  AND p.fechahorabajapermiso IS NULL
					  AND p.clavepermiso = 'GESTIONAR_CANCIONES'
			  )
		)
	`,
		codigoIntegrante,
		codigoProyecto,
	).Scan(&puede)

	return puede, err
}

func (r *proyectoRepository) PuedeRealizarEnProyecto(
	codigoIntegrante int64,
	codigoProyecto int64,
	clavePermiso string,
) (bool, error) {

	var puede bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM integranteproyecto ip
			WHERE ip.codintegrante = $1
			  AND ip.codigoproyecto = $2
			  AND ip.fechahorabajaintegranteproy IS NULL
			  AND EXISTS (
					SELECT 1
					FROM rolpermiso rp
					JOIN permiso p
					  ON p.codigopermiso = rp.codigopermiso
					WHERE rp.codrol = ip.codrol
					  AND rp.ambitorolpermiso = ip.ambitorol
					  AND rp.fechahorabajarolpermiso IS NULL
					  AND p.fechahorabajapermiso IS NULL
					  AND p.clavepermiso = $3
			  )
		)
	`,
		codigoIntegrante,
		codigoProyecto,
		clavePermiso,
	).Scan(&puede)

	return puede, err
}

func (r *proyectoRepository) EsIntegranteActivo(
	codigoIntegrante int64,
	codigoProyecto int64,
) (bool, error) {

	var esIntegrante bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM integranteproyecto
			WHERE codintegrante = $1
			  AND codigoproyecto = $2
			  AND fechahorabajaintegranteproy IS NULL
		)
	`,
		codigoIntegrante,
		codigoProyecto,
	).Scan(&esIntegrante)

	return esIntegrante, err
}

func (r *proyectoRepository) EsPropietarioActivo(
	codigoIntegrante int64,
	codigoProyecto int64,
) (bool, error) {

	var esPropietario bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM integranteproyecto
			WHERE codintegrante = $1
			  AND codigoproyecto = $2
			  AND espropietario = TRUE
			  AND fechahorabajaintegranteproy IS NULL
		)
	`,
		codigoIntegrante,
		codigoProyecto,
	).Scan(&esPropietario)

	return esPropietario, err
}

// Las consultas de "mis proyectos" se escriben completas y por separado, sin
// concatenar fragmentos, para que el análisis estático las reconozca como SQL
// constante. El FROM/WHERE debe ser idéntico en ambas (incluidos los JOIN,
// que también filtran filas) para que totalItems refleje lo que se pagina;
// TestConsultasMisProyectosCompartenFiltro lo verifica.
//
// En ambas: $1 integrante, $2 patrón de búsqueda ya escapado (cadena vacía =
// sin filtro), $3 códigos de estado y $4 códigos de tipo (NULL o arreglo
// vacío = sin filtro).

const consultaContarMisProyectos = `
		SELECT COUNT(*)
		FROM integranteproyecto ip
		JOIN proyecto p
		  ON p.codigoproyecto = ip.codigoproyecto
		JOIN tipoproyecto tp
		  ON tp.codtipoproy = p.codtipoproy
		JOIN estadoproyecto ep
		  ON ep.codestadoproy = p.codestadoproy
		JOIN rol r
		  ON r.codrol = ip.codrol
		 AND r.ambitorol = ip.ambitorol
		WHERE ip.codintegrante = $1
		  AND ip.fechahorabajaintegranteproy IS NULL
		  AND p.fechahorabajaproyecto IS NULL
		  AND (
			$2::text = ''
			OR p.nombreproyecto ILIKE '%' || $2::text || '%'
		  )
		  AND (
			COALESCE(array_length($3::bigint[], 1), 0) = 0
			OR p.codestadoproy = ANY($3::bigint[])
		  )
		  AND (
			COALESCE(array_length($4::bigint[], 1), 0) = 0
			OR p.codtipoproy = ANY($4::bigint[])
		  )
`

// La fecha de última modificación se deriva: el proyecto no guarda una. Es
// lo más reciente entre su alta (la del primer integrante, que es quien lo
// creó) y la última versión activa de alguna de sus canciones activas.
// GREATEST ignora el NULL de un proyecto sin versiones.
//
// Una canción está "lista" si su versión actual (la de mayor numeroversion
// sin baja) está en la última etapa activa del catálogo. Sin etapas, la
// subconsulta de la etapa da NULL y no cuenta ninguna.
const consultaListarMisProyectos = `
		SELECT
			p.codigoproyecto,
			p.nombreproyecto,
			p.descripcionproyecto,
			p.logoproyecto,
			tp.nombretipoproy,
			ep.nombreestadoproy,
			(
				SELECT COUNT(*)
				FROM cancion c
				WHERE c.codigoproyecto = p.codigoproyecto
				  AND c.fechahorabajacancion IS NULL
			) AS cantidadcanciones,
			(
				SELECT COUNT(*)
				FROM cancion c
				CROSS JOIN LATERAL (
					SELECT cv.codetapaversion
					FROM cancionversion cv
					WHERE cv.codigocancion = c.codigocancion
					  AND cv.fechahorabajaversion IS NULL
					ORDER BY cv.numeroversion DESC
					LIMIT 1
				) actual
				WHERE c.codigoproyecto = p.codigoproyecto
				  AND c.fechahorabajacancion IS NULL
				  AND actual.codetapaversion = (
					SELECT e.codetapaversion
					FROM etapaversionado e
					WHERE e.fechahorabajaetapaversion IS NULL
					ORDER BY e.ordenetapaversion DESC, e.codetapaversion DESC
					LIMIT 1
				  )
			) AS cancioneslistas,
			ip.codrol,
			r.nombrerol,
			ip.espropietario,
			actividad.fechaultimamodificacion
		FROM integranteproyecto ip
		JOIN proyecto p
		  ON p.codigoproyecto = ip.codigoproyecto
		JOIN tipoproyecto tp
		  ON tp.codtipoproy = p.codtipoproy
		JOIN estadoproyecto ep
		  ON ep.codestadoproy = p.codestadoproy
		JOIN rol r
		  ON r.codrol = ip.codrol
		 AND r.ambitorol = ip.ambitorol
		CROSS JOIN LATERAL (
			SELECT GREATEST(
				(
					SELECT MIN(alta.fechahoraaltaintegranteproy)
					FROM integranteproyecto alta
					WHERE alta.codigoproyecto = p.codigoproyecto
				),
				(
					SELECT MAX(cv.fechahoraaltaversion)
					FROM cancion c
					JOIN cancionversion cv
					  ON cv.codigocancion = c.codigocancion
					WHERE c.codigoproyecto = p.codigoproyecto
					  AND c.fechahorabajacancion IS NULL
					  AND cv.fechahorabajaversion IS NULL
				)
			) AS fechaultimamodificacion
		) actividad
		WHERE ip.codintegrante = $1
		  AND ip.fechahorabajaintegranteproy IS NULL
		  AND p.fechahorabajaproyecto IS NULL
		  AND (
			$2::text = ''
			OR p.nombreproyecto ILIKE '%' || $2::text || '%'
		  )
		  AND (
			COALESCE(array_length($3::bigint[], 1), 0) = 0
			OR p.codestadoproy = ANY($3::bigint[])
		  )
		  AND (
			COALESCE(array_length($4::bigint[], 1), 0) = 0
			OR p.codtipoproy = ANY($4::bigint[])
		  )
		ORDER BY actividad.fechaultimamodificacion DESC,
			p.codigoproyecto DESC
		LIMIT $5 OFFSET $6
`

// codigosComoArreglo adapta un filtro por códigos al bigint[] que esperan las
// consultas. Vacío viaja como NULL, que las consultas leen como "sin filtro".
func codigosComoArreglo(codigos []int64) any {
	if len(codigos) == 0 {
		return nil
	}

	return codigos
}

func (r *proyectoRepository) ContarPorIntegrante(
	codigoIntegrante int64,
	filtro dto.ListarProyectosFiltro,
) (int, error) {

	var total int

	err := r.db.QueryRow(
		consultaContarMisProyectos,
		codigoIntegrante,
		escaparPatronLike(filtro.Busqueda),
		codigosComoArreglo(filtro.Estados),
		codigosComoArreglo(filtro.Tipos),
	).Scan(&total)

	if err != nil {
		return 0, err
	}

	return total, nil
}

func (r *proyectoRepository) ListarPorIntegrante(
	codigoIntegrante int64,
	filtro dto.ListarProyectosFiltro,
	desplazamiento int,
) ([]dto.ProyectoListadoResponse, error) {

	rows, err := r.db.Query(
		consultaListarMisProyectos,
		codigoIntegrante,
		escaparPatronLike(filtro.Busqueda),
		codigosComoArreglo(filtro.Estados),
		codigosComoArreglo(filtro.Tipos),
		filtro.TamanoPagina,
		desplazamiento,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	proyectos := make(
		[]dto.ProyectoListadoResponse,
		0,
	)

	for rows.Next() {

		var proyecto dto.ProyectoListadoResponse

		err := rows.Scan(
			&proyecto.CodigoProyecto,
			&proyecto.Nombre,
			&proyecto.Descripcion,
			&proyecto.Logo,
			&proyecto.Tipo,
			&proyecto.Estado,
			&proyecto.CantidadCanciones,
			&proyecto.CancionesListas,
			&proyecto.CodRol,
			&proyecto.NombreRol,
			&proyecto.EsPropietario,
			&proyecto.FechaUltimaModificacion,
		)

		if err != nil {
			return nil, err
		}

		proyectos = append(
			proyectos,
			proyecto,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return proyectos, nil
}

func (r *proyectoRepository) ListarColaboradores(
	codigoProyecto int64,
) ([]model.ColaboradorProyecto, error) {

	rows, err := r.db.Query(`
		SELECT
			i.codintegrante,
			i.nombreintegrante,
			i.urlavatarintegrante,
			ip.codrol,
			r.nombrerol,
			ip.espropietario
		FROM integranteproyecto ip
		JOIN integrante i
		  ON i.codintegrante = ip.codintegrante
		JOIN rol r
		  ON r.codrol = ip.codrol
		 AND r.ambitorol = ip.ambitorol
		WHERE ip.codigoproyecto = $1
		  AND ip.fechahorabajaintegranteproy IS NULL
		  AND i.fechahorabajaintegrante IS NULL
		ORDER BY ip.espropietario DESC, i.nombreintegrante ASC
	`,
		codigoProyecto,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	colaboradores := make(
		[]model.ColaboradorProyecto,
		0,
	)

	for rows.Next() {

		var colaborador model.ColaboradorProyecto

		err := rows.Scan(
			&colaborador.CodIntegrante,
			&colaborador.NombreIntegrante,
			&colaborador.AvatarObjectKey,
			&colaborador.CodRol,
			&colaborador.NombreRol,
			&colaborador.EsPropietario,
		)

		if err != nil {
			return nil, err
		}

		colaboradores = append(
			colaboradores,
			colaborador,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return colaboradores, nil
}

func (r *proyectoRepository) ExisteEstadoProyectoActivo(
	codigoEstadoProyecto int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM estadoproyecto
			WHERE codestadoproy = $1
			  AND fechahorabajaestadoproy IS NULL
		)
	`,
		codigoEstadoProyecto,
	).Scan(&existe)

	return existe, err
}

func (r *proyectoRepository) ObtenerDetalle(
	codigoProyecto int64,
) (*model.Proyecto, string, string, error) {

	var proyecto model.Proyecto
	var nombreTipoProyecto string
	var nombreEstadoProyecto string

	err := r.db.QueryRow(`
		SELECT
			p.codigoproyecto,
			p.nombreproyecto,
			p.descripcionproyecto,
			p.logoproyecto,
			p.codestadoproy,
			p.codtipoproy,
			p.fechahorabajaproyecto,
			tp.nombretipoproy,
			ep.nombreestadoproy
		FROM proyecto p
		INNER JOIN tipoproyecto tp
			ON tp.codtipoproy = p.codtipoproy
		INNER JOIN estadoproyecto ep
			ON ep.codestadoproy = p.codestadoproy
		WHERE p.codigoproyecto = $1
		  AND p.fechahorabajaproyecto IS NULL
	`,
		codigoProyecto,
	).Scan(
		&proyecto.CodigoProyecto,
		&proyecto.NombreProyecto,
		&proyecto.DescripcionProyecto,
		&proyecto.LogoProyecto,
		&proyecto.CodEstadoProy,
		&proyecto.CodTipoProy,
		&proyecto.FechaHoraBajaProyecto,
		&nombreTipoProyecto,
		&nombreEstadoProyecto,
	)

	if err != nil {
		return nil, "", "", err
	}

	return &proyecto,
		nombreTipoProyecto,
		nombreEstadoProyecto,
		nil
}

func (r *proyectoRepository) ListarGenerosProyecto(
	codigoProyecto int64,
) ([]dto.GeneroProyectoResponse, error) {

	rows, err := r.db.Query(`
		SELECT
			g.codigogeneroproy,
			g.nombregeneroproy
		FROM proyectogeneromusical pg
		INNER JOIN generomusicalproyecto g
			ON g.codigogeneroproy = pg.codigogeneroproy
		WHERE pg.codigoproyecto = $1
		  AND g.fechahorabajageneroproy IS NULL
		ORDER BY g.nombregeneroproy
	`,
		codigoProyecto,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	generos := make(
		[]dto.GeneroProyectoResponse,
		0,
	)

	for rows.Next() {

		var genero dto.GeneroProyectoResponse

		if err := rows.Scan(
			&genero.CodigoGenero,
			&genero.NombreGenero,
		); err != nil {
			return nil, err
		}

		generos = append(
			generos,
			genero,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return generos, nil
}

func (r *proyectoRepository) Actualizar(
	codigoProyecto int64,
	nombre string,
	descripcion *string,
	cambiarLogo bool,
	logoObjectKey *string,
	codigoEstadoProyecto int64,
	codigoTipoProyecto int64,
	codigosGeneros []int64,
) (*string, *string, error) {

	tx, err := r.db.Begin()
	if err != nil {
		return nil, nil, err
	}

	defer tx.Rollback()

	// FOR UPDATE: dos ediciones simultáneas no pueden leer el mismo logo
	// anterior, así cada una borra solo el archivo que ella reemplazó.
	var logoAnterior *string

	err = tx.QueryRow(`
		SELECT logoproyecto
		FROM proyecto
		WHERE codigoproyecto = $1
		  AND fechahorabajaproyecto IS NULL
		FOR UPDATE
	`,
		codigoProyecto,
	).Scan(&logoAnterior)

	if err != nil {
		return nil, nil, err
	}

	// Sin cambio de logo se conserva el valor de la fila: no se reescribe
	// el que el service leyó antes, que pudo haber cambiado desde entonces.
	var logoVigente *string

	err = tx.QueryRow(`
		UPDATE proyecto
		SET
			nombreproyecto = $1,
			descripcionproyecto = $2,
			logoproyecto = CASE WHEN $3 THEN $4 ELSE logoproyecto END,
			codestadoproy = $5,
			codtipoproy = $6
		WHERE codigoproyecto = $7
		RETURNING logoproyecto
	`,
		nombre,
		descripcion,
		cambiarLogo,
		logoObjectKey,
		codigoEstadoProyecto,
		codigoTipoProyecto,
		codigoProyecto,
	).Scan(&logoVigente)

	if err != nil {
		return nil, nil, err
	}

	// Eliminamos las relaciones actuales.
	_, err = tx.Exec(`
		DELETE FROM proyectogeneromusical
		WHERE codigoproyecto = $1
	`,
		codigoProyecto,
	)

	if err != nil {
		return nil, nil, err
	}

	// Insertamos nuevamente los géneros seleccionados.
	for _, codigoGenero := range codigosGeneros {

		_, err = tx.Exec(`
			INSERT INTO proyectogeneromusical (
				codigoproyecto,
				codigogeneroproy
			)
			VALUES ($1, $2)
		`,
			codigoProyecto,
			codigoGenero,
		)

		if err != nil {
			return nil, nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrConfirmacionIncierta, err)
	}

	if !cambiarLogo {
		return logoVigente, nil, nil
	}

	return logoVigente, logoAnterior, nil
}

func (r *proyectoRepository) DarDeBaja(
	codigoProyecto int64,
) error {

	resultado, err := r.db.Exec(`
		UPDATE proyecto
		SET fechahorabajaproyecto = CURRENT_TIMESTAMP
		WHERE codigoproyecto = $1
		  AND fechahorabajaproyecto IS NULL
	`,
		codigoProyecto,
	)

	if err != nil {
		return err
	}

	filas, err := resultado.RowsAffected()
	if err != nil {
		return err
	}

	if filas == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *proyectoRepository) ListarMiParticipacion(
	codigoIntegrante int64,
	participacion string,
) ([]dto.MiParticipacionProyectoResponse, error) {

	rows, err := r.db.Query(`
		SELECT
			p.codigoproyecto,
			p.nombreproyecto,
			ip.codrol,
			r.nombrerol,
			ip.espropietario
		FROM integranteproyecto ip
		INNER JOIN proyecto p
			ON p.codigoproyecto = ip.codigoproyecto
		INNER JOIN rol r
			ON r.codrol = ip.codrol
		   AND r.ambitorol = ip.ambitorol
		WHERE ip.codintegrante = $1
		  AND ip.fechahorabajaintegranteproy IS NULL
		  AND p.fechahorabajaproyecto IS NULL
		  AND r.fechahorabajarol IS NULL
		  AND r.ambitorol = 'PROYECTO'
		  AND (
				$2 = 'todos'

				OR (
					$2 = 'propietario'
					AND ip.espropietario = TRUE
				)

				OR (
					$2 = 'productor'
					AND LOWER(r.nombrerol) =
						LOWER($3)
				)

				OR (
					$2 = 'artista'
					AND LOWER(r.nombrerol) =
						LOWER($4)
				)
		  )
		ORDER BY p.nombreproyecto
	`,
		codigoIntegrante,
		participacion,
		model.NombreRolProductor,
		model.NombreRolMusicoArtista,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	proyectos := make(
		[]dto.MiParticipacionProyectoResponse,
		0,
	)

	for rows.Next() {

		var proyecto dto.MiParticipacionProyectoResponse

		err := rows.Scan(
			&proyecto.CodigoProyecto,
			&proyecto.Nombre,
			&proyecto.CodRol,
			&proyecto.NombreRol,
			&proyecto.EsPropietario,
		)

		if err != nil {
			return nil, err
		}

		proyectos = append(
			proyectos,
			proyecto,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return proyectos, nil
}

func (r *proyectoRepository) ObtenerColaboradorActivo(
	codigoProyecto int64,
	codigoIntegrante int64,
) (*model.ColaboradorProyecto, error) {

	var colaborador model.ColaboradorProyecto

	err := r.db.QueryRow(`
		SELECT
			i.codintegrante,
			i.nombreintegrante,
			i.urlavatarintegrante,
			ip.codrol,
			r.nombrerol,
			ip.espropietario
		FROM integranteproyecto ip
		JOIN integrante i
		  ON i.codintegrante = ip.codintegrante
		JOIN rol r
		  ON r.codrol = ip.codrol
		 AND r.ambitorol = ip.ambitorol
		WHERE ip.codigoproyecto = $1
		  AND ip.codintegrante = $2
		  AND ip.fechahorabajaintegranteproy IS NULL
		  AND i.fechahorabajaintegrante IS NULL
	`,
		codigoProyecto,
		codigoIntegrante,
	).Scan(
		&colaborador.CodIntegrante,
		&colaborador.NombreIntegrante,
		&colaborador.AvatarObjectKey,
		&colaborador.CodRol,
		&colaborador.NombreRol,
		&colaborador.EsPropietario,
	)

	if err != nil {
		return nil, err
	}

	return &colaborador, nil
}

func (r *proyectoRepository) ActualizarRolIntegrante(
	codigoProyecto int64,
	codigoIntegrante int64,
	codRol int64,
) error {

	resultado, err := r.db.Exec(`
		UPDATE integranteproyecto
		SET codrol = $3
		WHERE codigoproyecto = $1
		  AND codintegrante = $2
		  AND fechahorabajaintegranteproy IS NULL
	`,
		codigoProyecto,
		codigoIntegrante,
		codRol,
	)

	if err != nil {
		return err
	}

	filas, err := resultado.RowsAffected()

	if err != nil {
		return err
	}

	if filas == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *proyectoRepository) DarDeBajaIntegrante(
	codigoProyecto int64,
	codigoIntegrante int64,
) error {

	resultado, err := r.db.Exec(`
		UPDATE integranteproyecto
		SET fechahorabajaintegranteproy = CURRENT_TIMESTAMP
		WHERE codigoproyecto = $1
		  AND codintegrante = $2
		  AND fechahorabajaintegranteproy IS NULL
		  AND espropietario = FALSE
	`,
		codigoProyecto,
		codigoIntegrante,
	)

	if err != nil {
		return err
	}

	filas, err := resultado.RowsAffected()

	if err != nil {
		return err
	}

	if filas == 0 {
		return sql.ErrNoRows
	}

	return nil
}
