package repository

import (
	"database/sql"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

// Tablero (HU-DASH-B01/B02). Todas las consultas reciben $1 integrante y $2
// proyecto (NULL = todos los proyectos del integrante). El alcance son los
// proyectos en los que el integrante participa hoy (membresía activa).
//
// Las consultas se escriben como constantes (sin concatenar en tiempo de
// ejecución) para que el análisis estático las reconozca como SQL fijo.

// TableroConteos son los valores crudos de los indicadores: el valor actual
// y el que tenía cada uno al inicio del período, para calcular la variación.
type TableroConteos struct {
	ProyectosActuales int64
	ProyectosInicio   int64

	CancionesActuales int64
	CancionesInicio   int64

	VersionesActuales int64
	VersionesInicio   int64

	// Versiones subidas en el período y en el período anterior.
	VersionesPeriodo         int64
	VersionesPeriodoAnterior int64

	PendientesActuales int64
	PendientesInicio   int64

	StemsActuales int64
	StemsInicio   int64
}

// TableroEtapaConteo es la cantidad de canciones cuya versión actual está en
// la etapa. Activa = la etapa no está dada de baja en el catálogo.
type TableroEtapaConteo struct {
	Etapa    string
	Cantidad int64
	Activa   bool
}

type TableroRepository interface {
	ObtenerConteos(
		codigoIntegrante int64,
		codigoProyecto *int64,
		inicioPeriodo time.Time,
		ahora time.Time,
		inicioPeriodoAnterior time.Time,
	) (TableroConteos, error)

	ListarVersionesPorProyecto(
		codigoIntegrante int64,
		codigoProyecto *int64,
	) ([]dto.TableroVersionesPorProyectoResponse, error)

	// ContarCancionesPorEtapa devuelve el conteo de cada etapa del catálogo
	// y el total de canciones del alcance (las que no suman a ninguna etapa
	// no tienen etapa asignada).
	ContarCancionesPorEtapa(
		codigoIntegrante int64,
		codigoProyecto *int64,
	) ([]TableroEtapaConteo, int64, error)

	// ContarActividad agrupa versiones y comentarios en los períodos
	// delimitados por limites (n+1 límites = n períodos). Devuelve un mapa
	// índice de período (1..n) → conteos; los períodos sin actividad no
	// aparecen.
	ContarActividad(
		codigoIntegrante int64,
		codigoProyecto *int64,
		limites []time.Time,
	) (map[int]TableroActividadConteo, error)

	// ListarNovedades devuelve las últimas limite novedades de los proyectos
	// del integrante y cuántas hay posteriores a desde (0 si desde es nil).
	ListarNovedades(
		codigoIntegrante int64,
		desde *time.Time,
		limite int,
	) ([]dto.TableroNovedadResponse, int64, error)

	// RegistrarVisita guarda ahora como última visita del usuario y devuelve
	// la que tenía antes (nil si nunca había entrado).
	RegistrarVisita(codigoUsuario int64, ahora time.Time) (*time.Time, error)
}

type TableroActividadConteo struct {
	Versiones   int64
	Comentarios int64
}

type tableroRepository struct {
	db *sql.DB
}

func NewTableroRepository(db *sql.DB) TableroRepository {
	return &tableroRepository{db: db}
}

// Los indicadores comparan contra el inicio del período ($3), así que no
// pueden filtrar las bajas de entrada: cada CTE arrastra su fecha de alta y
// la de baja efectiva (la propia o la de lo que la contiene; LEAST ignora
// los NULL). Algo "existía" en un instante T si alta <= T y no tenía baja
// antes de T.
//
// Fechas de alta derivadas, porque no hay columna: la del proyecto es la del
// primer integrante (quien lo creó) y la de la canción, la de su primera
// versión.
//
// Los comentarios pendientes al inicio del período son una aproximación: no
// se guarda el historial de cambios de estado, así que se cuentan los que
// existían en ese momento y hoy siguen pendientes.
const consultaConteosTablero = `
		WITH proyectos AS (
			SELECT DISTINCT
				p.codigoproyecto,
				p.fechahorabajaproyecto AS baja,
				(
					SELECT MIN(alta.fechahoraaltaintegranteproy)
					FROM integranteproyecto alta
					WHERE alta.codigoproyecto = p.codigoproyecto
				) AS alta
			FROM integranteproyecto ip
			JOIN proyecto p
			  ON p.codigoproyecto = ip.codigoproyecto
			WHERE ip.codintegrante = $1
			  AND ip.fechahorabajaintegranteproy IS NULL
			  AND ($2::bigint IS NULL OR p.codigoproyecto = $2::bigint)
		),
		canciones AS (
			SELECT
				c.codigocancion,
				LEAST(c.fechahorabajacancion, pr.baja) AS baja,
				(
					SELECT MIN(primera.fechahoraaltaversion)
					FROM cancionversion primera
					WHERE primera.codigocancion = c.codigocancion
				) AS alta
			FROM cancion c
			JOIN proyectos pr
			  ON pr.codigoproyecto = c.codigoproyecto
		),
		versiones AS (
			SELECT
				cv.codigocancionversion,
				cv.fechahoraaltaversion AS alta,
				LEAST(cv.fechahorabajaversion, ca.baja) AS baja
			FROM cancionversion cv
			JOIN canciones ca
			  ON ca.codigocancion = cv.codigocancion
		),
		comentarios AS (
			SELECT
				co.fechahoraaltacomentario AS alta,
				LEAST(co.fechahorabajacomentario, v.baja) AS baja,
				LOWER(ec.nombreestadocom) = 'pendiente' AS pendiente
			FROM comentario co
			JOIN versiones v
			  ON v.codigocancionversion = co.codigocancionversion
			JOIN estadocomentario ec
			  ON ec.codestadocom = co.codestadocom
		),
		stems AS (
			SELECT
				s.fechahoraaltastem AS alta,
				LEAST(s.fechahorabajastem, v.baja) AS baja
			FROM stem s
			JOIN versiones v
			  ON v.codigocancionversion = s.codigocancionversion
		)
		SELECT
			(SELECT COUNT(*) FROM proyectos WHERE baja IS NULL),
			(SELECT COUNT(*) FROM proyectos
			  WHERE alta <= $3 AND (baja IS NULL OR baja > $3)),
			(SELECT COUNT(*) FROM canciones WHERE baja IS NULL),
			(SELECT COUNT(*) FROM canciones
			  WHERE alta <= $3 AND (baja IS NULL OR baja > $3)),
			(SELECT COUNT(*) FROM versiones WHERE baja IS NULL),
			(SELECT COUNT(*) FROM versiones
			  WHERE alta <= $3 AND (baja IS NULL OR baja > $3)),
			(SELECT COUNT(*) FROM versiones
			  WHERE baja IS NULL AND alta > $3 AND alta <= $4),
			(SELECT COUNT(*) FROM versiones
			  WHERE baja IS NULL AND alta > $5 AND alta <= $3),
			(SELECT COUNT(*) FROM comentarios WHERE pendiente AND baja IS NULL),
			(SELECT COUNT(*) FROM comentarios
			  WHERE pendiente AND alta <= $3 AND (baja IS NULL OR baja > $3)),
			(SELECT COUNT(*) FROM stems WHERE baja IS NULL),
			(SELECT COUNT(*) FROM stems
			  WHERE alta <= $3 AND (baja IS NULL OR baja > $3))
`

func (r *tableroRepository) ObtenerConteos(
	codigoIntegrante int64,
	codigoProyecto *int64,
	inicioPeriodo time.Time,
	ahora time.Time,
	inicioPeriodoAnterior time.Time,
) (TableroConteos, error) {

	var conteos TableroConteos

	err := r.db.QueryRow(
		consultaConteosTablero,
		codigoIntegrante,
		codigoProyecto,
		inicioPeriodo,
		ahora,
		inicioPeriodoAnterior,
	).Scan(
		&conteos.ProyectosActuales,
		&conteos.ProyectosInicio,
		&conteos.CancionesActuales,
		&conteos.CancionesInicio,
		&conteos.VersionesActuales,
		&conteos.VersionesInicio,
		&conteos.VersionesPeriodo,
		&conteos.VersionesPeriodoAnterior,
		&conteos.PendientesActuales,
		&conteos.PendientesInicio,
		&conteos.StemsActuales,
		&conteos.StemsInicio,
	)

	return conteos, err
}

// Los gráficos muestran el estado actual: proyectos, canciones y versiones
// sin baja. Los comentarios cuentan aunque sean de un stem (siguen
// perteneciendo a la versión).

const consultaVersionesPorProyectoTablero = `
		SELECT
			p.codigoproyecto,
			p.nombreproyecto,
			COUNT(cv.codigocancionversion) AS cantidadversiones
		FROM proyecto p
		JOIN cancion c
		  ON c.codigoproyecto = p.codigoproyecto
		 AND c.fechahorabajacancion IS NULL
		JOIN cancionversion cv
		  ON cv.codigocancion = c.codigocancion
		 AND cv.fechahorabajaversion IS NULL
		WHERE p.fechahorabajaproyecto IS NULL
		  AND ($2::bigint IS NULL OR p.codigoproyecto = $2::bigint)
		  AND EXISTS (
				SELECT 1
				FROM integranteproyecto ip
				WHERE ip.codigoproyecto = p.codigoproyecto
				  AND ip.codintegrante = $1
				  AND ip.fechahorabajaintegranteproy IS NULL
		  )
		GROUP BY p.codigoproyecto, p.nombreproyecto
		ORDER BY cantidadversiones DESC, p.nombreproyecto, p.codigoproyecto
`

func (r *tableroRepository) ListarVersionesPorProyecto(
	codigoIntegrante int64,
	codigoProyecto *int64,
) ([]dto.TableroVersionesPorProyectoResponse, error) {

	rows, err := r.db.Query(
		consultaVersionesPorProyectoTablero,
		codigoIntegrante,
		codigoProyecto,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	resultado := make([]dto.TableroVersionesPorProyectoResponse, 0)

	for rows.Next() {

		var fila dto.TableroVersionesPorProyectoResponse

		if err := rows.Scan(
			&fila.ProyectoID,
			&fila.Nombre,
			&fila.CantidadVersiones,
		); err != nil {
			return nil, err
		}

		resultado = append(resultado, fila)
	}

	return resultado, rows.Err()
}

// La etapa de una canción es la de su versión actual (la de mayor
// numeroversion sin baja). Se listan todas las etapas del catálogo, incluso
// las dadas de baja, para que el service decida cuáles mostrar; el total de
// canciones viaja en cada fila (ventana) para no hacer otra consulta.
const consultaCancionesPorEtapaTablero = `
		WITH canciones AS (
			SELECT c.codigocancion
			FROM cancion c
			JOIN proyecto p
			  ON p.codigoproyecto = c.codigoproyecto
			WHERE c.fechahorabajacancion IS NULL
			  AND p.fechahorabajaproyecto IS NULL
			  AND ($2::bigint IS NULL OR p.codigoproyecto = $2::bigint)
			  AND EXISTS (
					SELECT 1
					FROM integranteproyecto ip
					WHERE ip.codigoproyecto = p.codigoproyecto
					  AND ip.codintegrante = $1
					  AND ip.fechahorabajaintegranteproy IS NULL
			  )
		),
		actuales AS (
			SELECT DISTINCT ON (cv.codigocancion)
				cv.codigocancion,
				cv.codetapaversion
			FROM cancionversion cv
			JOIN canciones c
			  ON c.codigocancion = cv.codigocancion
			WHERE cv.fechahorabajaversion IS NULL
			ORDER BY cv.codigocancion, cv.numeroversion DESC
		)
		SELECT
			e.nombreetapaversion,
			COUNT(a.codigocancion),
			e.fechahorabajaetapaversion IS NULL,
			(SELECT COUNT(*) FROM canciones)
		FROM etapaversionado e
		LEFT JOIN actuales a
		  ON a.codetapaversion = e.codetapaversion
		GROUP BY
			e.codetapaversion,
			e.nombreetapaversion,
			e.ordenetapaversion,
			e.fechahorabajaetapaversion
		ORDER BY e.ordenetapaversion, e.nombreetapaversion
`

// Sin etapas en el catálogo la consulta de arriba no devuelve filas y no
// trae el total: se pide aparte.
const consultaTotalCancionesTablero = `
		SELECT COUNT(*)
		FROM cancion c
		JOIN proyecto p
		  ON p.codigoproyecto = c.codigoproyecto
		WHERE c.fechahorabajacancion IS NULL
		  AND p.fechahorabajaproyecto IS NULL
		  AND ($2::bigint IS NULL OR p.codigoproyecto = $2::bigint)
		  AND EXISTS (
				SELECT 1
				FROM integranteproyecto ip
				WHERE ip.codigoproyecto = p.codigoproyecto
				  AND ip.codintegrante = $1
				  AND ip.fechahorabajaintegranteproy IS NULL
		  )
`

func (r *tableroRepository) ContarCancionesPorEtapa(
	codigoIntegrante int64,
	codigoProyecto *int64,
) ([]TableroEtapaConteo, int64, error) {

	rows, err := r.db.Query(
		consultaCancionesPorEtapaTablero,
		codigoIntegrante,
		codigoProyecto,
	)

	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	etapas := make([]TableroEtapaConteo, 0)
	var totalCanciones int64

	for rows.Next() {

		var etapa TableroEtapaConteo

		if err := rows.Scan(
			&etapa.Etapa,
			&etapa.Cantidad,
			&etapa.Activa,
			&totalCanciones,
		); err != nil {
			return nil, 0, err
		}

		etapas = append(etapas, etapa)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(etapas) == 0 {
		err = r.db.QueryRow(
			consultaTotalCancionesTablero,
			codigoIntegrante,
			codigoProyecto,
		).Scan(&totalCanciones)

		if err != nil {
			return nil, 0, err
		}
	}

	return etapas, totalCanciones, nil
}

// width_bucket ubica cada fecha en el período [limites[i-1], limites[i]) y
// devuelve i. Los límites los arma el service en hora local, así la semana o
// el mes empiezan donde los ve el usuario y no en UTC.
const consultaActividadTablero = `
		WITH versiones AS (
			SELECT
				cv.codigocancionversion,
				cv.fechahoraaltaversion
			FROM cancionversion cv
			JOIN cancion c
			  ON c.codigocancion = cv.codigocancion
			 AND c.fechahorabajacancion IS NULL
			JOIN proyecto p
			  ON p.codigoproyecto = c.codigoproyecto
			 AND p.fechahorabajaproyecto IS NULL
			WHERE cv.fechahorabajaversion IS NULL
			  AND ($2::bigint IS NULL OR p.codigoproyecto = $2::bigint)
			  AND EXISTS (
					SELECT 1
					FROM integranteproyecto ip
					WHERE ip.codigoproyecto = p.codigoproyecto
					  AND ip.codintegrante = $1
					  AND ip.fechahorabajaintegranteproy IS NULL
			  )
		),
		actividad AS (
			SELECT v.fechahoraaltaversion AS fecha, TRUE AS esversion
			FROM versiones v
			UNION ALL
			SELECT co.fechahoraaltacomentario, FALSE
			FROM comentario co
			JOIN versiones v
			  ON v.codigocancionversion = co.codigocancionversion
			WHERE co.fechahorabajacomentario IS NULL
		)
		SELECT
			width_bucket(fecha, $3::timestamptz[]) AS periodo,
			COUNT(*) FILTER (WHERE esversion),
			COUNT(*) FILTER (WHERE NOT esversion)
		FROM actividad
		WHERE fecha >= $4
		  AND fecha < $5
		GROUP BY periodo
`

func (r *tableroRepository) ContarActividad(
	codigoIntegrante int64,
	codigoProyecto *int64,
	limites []time.Time,
) (map[int]TableroActividadConteo, error) {

	conteos := make(map[int]TableroActividadConteo)

	if len(limites) < 2 {
		return conteos, nil
	}

	rows, err := r.db.Query(
		consultaActividadTablero,
		codigoIntegrante,
		codigoProyecto,
		limites,
		limites[0],
		limites[len(limites)-1],
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {

		var periodo int
		var conteo TableroActividadConteo

		if err := rows.Scan(
			&periodo,
			&conteo.Versiones,
			&conteo.Comentarios,
		); err != nil {
			return nil, err
		}

		conteos[periodo] = conteo
	}

	return conteos, rows.Err()
}

// Novedades del Inicio: versiones, comentarios (también los de stems),
// respuestas, stems subidos a mano y separaciones con IA terminadas, de los
// proyectos, canciones y versiones activos donde el integrante ($1) participa
// hoy. Se excluye lo que hizo él mismo, salvo las separaciones (son el aviso
// de que terminó un proceso en segundo plano). Lo que no tiene autor
// (versiones y stems anteriores a que se guardara) no aparece.
//
// Los stems subidos a mano se agrupan por autor, versión y día: subir cinco
// pistas juntas es una sola novedad.
//
// novedadesdesde ($2) se cuenta con una ventana, que se calcula antes del
// LIMIT ($3): el total no depende de cuántas se piden.
const consultaNovedadesTablero = `
		WITH versiones AS (
			SELECT
				cv.codigocancionversion,
				cv.fechahoraaltaversion,
				cv.codintegrantealtaversion,
				c.codigocancion,
				c.nombrecancion,
				p.codigoproyecto,
				p.nombreproyecto
			FROM cancionversion cv
			JOIN cancion c
			  ON c.codigocancion = cv.codigocancion
			 AND c.fechahorabajacancion IS NULL
			JOIN proyecto p
			  ON p.codigoproyecto = c.codigoproyecto
			 AND p.fechahorabajaproyecto IS NULL
			WHERE cv.fechahorabajaversion IS NULL
			  AND EXISTS (
					SELECT 1
					FROM integranteproyecto ip
					WHERE ip.codigoproyecto = p.codigoproyecto
					  AND ip.codintegrante = $1
					  AND ip.fechahorabajaintegranteproy IS NULL
			  )
		),
		novedades AS (
			SELECT
				'VERSION' AS tipo,
				v.fechahoraaltaversion AS fecha,
				v.codintegrantealtaversion AS autor,
				v.codigocancionversion,
				1::bigint AS cantidad
			FROM versiones v
			UNION ALL
			SELECT 'COMENTARIO', co.fechahoraaltacomentario, co.codintegrante, v.codigocancionversion, 1
			FROM comentario co
			JOIN versiones v
			  ON v.codigocancionversion = co.codigocancionversion
			WHERE co.fechahorabajacomentario IS NULL
			UNION ALL
			SELECT 'RESPUESTA', r.fechahoraaltarespuesta, r.codintegrante, v.codigocancionversion, 1
			FROM comentariorespuesta r
			JOIN comentario co
			  ON co.codigocomentario = r.codigocomentario
			 AND co.fechahorabajacomentario IS NULL
			JOIN versiones v
			  ON v.codigocancionversion = co.codigocancionversion
			WHERE r.fechahorabajarespuesta IS NULL
			UNION ALL
			SELECT 'STEMS', MAX(s.fechahoraaltastem), s.codintegrantealtastem, v.codigocancionversion, COUNT(*)
			FROM stem s
			JOIN versiones v
			  ON v.codigocancionversion = s.codigocancionversion
			WHERE s.fechahorabajastem IS NULL
			  AND NOT s.generadoconia
			GROUP BY s.codintegrantealtastem, v.codigocancionversion, date_trunc('day', s.fechahoraaltastem)
			UNION ALL
			SELECT 'SEPARACION', sep.fechahorafin, sep.codintegrante, v.codigocancionversion, sep.cantidadstems
			FROM separacionstem sep
			JOIN versiones v
			  ON v.codigocancionversion = sep.codigocancionversion
			WHERE sep.estadoseparacion = 'COMPLETADA'
			  AND sep.fechahorafin IS NOT NULL
		)
		SELECT
			n.tipo,
			n.fecha,
			i.codintegrante,
			i.nombreintegrante,
			v.codigoproyecto,
			v.nombreproyecto,
			v.codigocancion,
			v.nombrecancion,
			n.cantidad,
			COUNT(*) FILTER (WHERE n.fecha > $2::timestamptz) OVER () AS novedadesdesde
		FROM novedades n
		JOIN integrante i
		  ON i.codintegrante = n.autor
		JOIN versiones v
		  ON v.codigocancionversion = n.codigocancionversion
		WHERE n.autor <> $1
		   OR n.tipo = 'SEPARACION'
		ORDER BY n.fecha DESC, n.tipo
		LIMIT $3
`

func (r *tableroRepository) ListarNovedades(
	codigoIntegrante int64,
	desde *time.Time,
	limite int,
) ([]dto.TableroNovedadResponse, int64, error) {

	rows, err := r.db.Query(
		consultaNovedadesTablero,
		codigoIntegrante,
		desde,
		limite,
	)

	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	novedades := make([]dto.TableroNovedadResponse, 0)
	var novedadesDesde int64

	for rows.Next() {

		var novedad dto.TableroNovedadResponse

		if err := rows.Scan(
			&novedad.Tipo,
			&novedad.Fecha,
			&novedad.Autor.CodigoIntegrante,
			&novedad.Autor.Nombre,
			&novedad.CodigoProyecto,
			&novedad.NombreProyecto,
			&novedad.CodigoCancion,
			&novedad.NombreCancion,
			&novedad.Cantidad,
			&novedadesDesde,
		); err != nil {
			return nil, 0, err
		}

		novedades = append(novedades, novedad)
	}

	return novedades, novedadesDesde, rows.Err()
}

// La última visita se guarda en usuario.ultimologin, que existía sin que
// nada la escribiera. El CTE lee el valor anterior y bloquea la fila
// (FOR UPDATE): dos visitas superpuestas se serializan y la segunda lee lo que
// escribió la primera. GREATEST evita que una request demorada, con un
// "ahora" viejo, retroceda la marca.
const consultaRegistrarVisita = `
		WITH anterior AS (
			SELECT ultimologin
			FROM usuario
			WHERE codigousuario = $1
			FOR UPDATE
		)
		UPDATE usuario
		SET ultimologin = GREATEST(ultimologin, $2)
		WHERE codigousuario = $1
		RETURNING (SELECT ultimologin FROM anterior)
`

func (r *tableroRepository) RegistrarVisita(codigoUsuario int64, ahora time.Time) (*time.Time, error) {

	var anterior sql.NullTime

	if err := r.db.QueryRow(consultaRegistrarVisita, codigoUsuario, ahora).Scan(&anterior); err != nil {
		return nil, err
	}

	if !anterior.Valid {
		return nil, nil
	}

	return &anterior.Time, nil
}
