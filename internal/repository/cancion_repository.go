package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type CancionRepository interface {
	ExisteNombreEnProyecto(
		codigoProyecto int64,
		nombre string,
	) (bool, error)

	IniciarCreacionCancion(
		codigoProyecto int64,
		nombre string,
	) (tx *sql.Tx, codigoCancion int64, err error)

	FinalizarCreacionVersionInicial(
		tx *sql.Tx,
		codigoCancion int64,
		urlArchivo string,
		formatoArchivo string,
	) (int64, error)

	ExisteCancionActivaEnProyecto(
		codigoProyecto int64,
		codigoCancion int64,
	) (bool, error)

	IniciarCreacionVersion(
		codigoCancion int64,
	) (tx *sql.Tx, siguienteVersion int, err error)

	InsertarVersion(
		tx *sql.Tx,
		codigoCancion int64,
		numeroVersion int,
		urlArchivo string,
		formatoArchivo string,
		notas *string,
	) (int64, error)

	InsertarStem(
		tx *sql.Tx,
		codigoCancionVersion int64,
		nombre string,
		urlArchivo string,
		formato string,
	) (int64, error)

	ExisteVersionActivaEnCancion(
		codigoCancion int64,
		codigoVersion int64,
	) (bool, error)

	BuscarVersionPorCodigo(
		codigoCancion int64,
		codigoCancionVersion int64,
	) (*model.CancionVersion, error)

	ListarPorProyecto(
		codigoProyecto int64,
	) ([]dto.CancionListadoResponse, error)

	ListarVersiones(
		codigoCancion int64,
	) ([]dto.VersionCancionListadoResponse, error)

	ContarPorIntegrante(
		codigoIntegrante int64,
		busqueda string,
	) (int, error)

	ListarPorIntegrante(
		codigoIntegrante int64,
		busqueda string,
		limite int,
		desplazamiento int,
	) ([]dto.MiCancionListadoResponse, error)
}

type cancionRepository struct {
	db *sql.DB
}

func NewCancionRepository(db *sql.DB) CancionRepository {
	return &cancionRepository{
		db: db,
	}
}

func (r *cancionRepository) ExisteNombreEnProyecto(
	codigoProyecto int64,
	nombre string,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM cancion
			WHERE codigoproyecto = $1
			  AND LOWER(TRIM(nombrecancion)) = LOWER(TRIM($2))
			  AND fechahorabajacancion IS NULL
		)
	`,
		codigoProyecto,
		nombre,
	).Scan(&existe)

	return existe, err
}

func (r *cancionRepository) IniciarCreacionCancion(
	codigoProyecto int64,
	nombre string,
) (*sql.Tx, int64, error) {

	tx, err := r.db.Begin()

	if err != nil {
		return nil, 0, err
	}

	var codigoCancion int64

	err = tx.QueryRow(`
		INSERT INTO cancion (
			codigoproyecto,
			nombrecancion
		)
		VALUES ($1, $2)
		RETURNING codigocancion
	`,
		codigoProyecto,
		nombre,
	).Scan(&codigoCancion)

	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	return tx, codigoCancion, nil
}

func (r *cancionRepository) FinalizarCreacionVersionInicial(
	tx *sql.Tx,
	codigoCancion int64,
	urlArchivo string,
	formatoArchivo string,
) (int64, error) {

	defer tx.Rollback()

	var codigoCancionVersion int64

	err := tx.QueryRow(`
		INSERT INTO cancionversion (
			codigocancion,
			numeroversion,
			fechahoraaltaversion,
			urlarchivocancionver,
			formatoarchivocancionver
		)
		VALUES (
			$1,
			1,
			CURRENT_TIMESTAMP,
			$2,
			$3
		)
		RETURNING codigocancionversion
	`,
		codigoCancion,
		urlArchivo,
		formatoArchivo,
	).Scan(&codigoCancionVersion)

	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return codigoCancionVersion, nil
}

func (r *cancionRepository) ExisteCancionActivaEnProyecto(
	codigoProyecto int64,
	codigoCancion int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM cancion
			WHERE codigocancion = $1
			  AND codigoproyecto = $2
			  AND fechahorabajacancion IS NULL
		)
	`,
		codigoCancion,
		codigoProyecto,
	).Scan(&existe)

	return existe, err
}

func (r *cancionRepository) IniciarCreacionVersion(
	codigoCancion int64,
) (*sql.Tx, int, error) {

	tx, err := r.db.Begin()

	if err != nil {
		return nil, 0, err
	}

	var codigoCancionBloqueada int64

	err = tx.QueryRow(`
		SELECT codigocancion
		FROM cancion
		WHERE codigocancion = $1
		  AND fechahorabajacancion IS NULL
		FOR UPDATE
	`,
		codigoCancion,
	).Scan(&codigoCancionBloqueada)

	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	var siguienteVersion int

	err = tx.QueryRow(`
		SELECT COALESCE(MAX(numeroversion), 0) + 1
		FROM cancionversion
		WHERE codigocancion = $1
		  AND fechahorabajaversion IS NULL
	`,
		codigoCancion,
	).Scan(&siguienteVersion)

	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	return tx, siguienteVersion, nil
}

// InsertarVersion inserta la fila de cancionversion dentro de la
// transacción abierta por IniciarCreacionVersion, sin commitear — el caller
// (el service) controla el commit/rollback una vez que también subió y
// registró los stems opcionales, para que todo quede atómico.
func (r *cancionRepository) InsertarVersion(
	tx *sql.Tx,
	codigoCancion int64,
	numeroVersion int,
	urlArchivo string,
	formatoArchivo string,
	notas *string,
) (int64, error) {

	var codigoCancionVersion int64

	err := tx.QueryRow(`
		INSERT INTO cancionversion (
			codigocancion,
			numeroversion,
			fechahoraaltaversion,
			urlarchivocancionver,
			formatoarchivocancionver,
			notasversion
		)
		VALUES (
			$1,
			$2,
			CURRENT_TIMESTAMP,
			$3,
			$4,
			$5
		)
		RETURNING codigocancionversion
	`,
		codigoCancion,
		numeroVersion,
		urlArchivo,
		formatoArchivo,
		notas,
	).Scan(&codigoCancionVersion)

	if err != nil {
		return 0, err
	}

	return codigoCancionVersion, nil
}

// InsertarStem inserta un stem opcional dentro de la misma transacción que
// InsertarVersion. El archivo se guarda en la columna correspondiente a su
// formato (wav o mp3) — la tabla stem no tiene columna de "formato" propia,
// a diferencia de cancionversion.
func (r *cancionRepository) InsertarStem(
	tx *sql.Tx,
	codigoCancionVersion int64,
	nombre string,
	urlArchivo string,
	formato string,
) (int64, error) {

	var codStem int64

	var urlWav, urlMp3 *string

	switch formato {
	case "wav":
		urlWav = &urlArchivo
	default:
		urlMp3 = &urlArchivo
	}

	err := tx.QueryRow(`
		INSERT INTO stem (
			codigocancionversion,
			nombrestem,
			urlversionwav,
			urlversionmp3
		)
		VALUES (
			$1,
			$2,
			$3,
			$4
		)
		RETURNING codstem
	`,
		codigoCancionVersion,
		nombre,
		urlWav,
		urlMp3,
	).Scan(&codStem)

	if err != nil {
		return 0, err
	}

	return codStem, nil
}

func (r *cancionRepository) ExisteVersionActivaEnCancion(
	codigoCancion int64,
	codigoVersion int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM cancionversion
			WHERE codigocancionversion = $1
			  AND codigocancion = $2
			  AND fechahorabajaversion IS NULL
		)
	`,
		codigoVersion,
		codigoCancion,
	).Scan(&existe)

	return existe, err
}

func (r *cancionRepository) BuscarVersionPorCodigo(
	codigoCancion int64,
	codigoCancionVersion int64,
) (*model.CancionVersion, error) {

	var version model.CancionVersion

	var urlArchivo sql.NullString
	var formatoArchivo sql.NullString
	var notas sql.NullString

	err := r.db.QueryRow(`
		SELECT
			codigocancionversion,
			codigocancion,
			numeroversion,
			fechahoraaltaversion,
			urlarchivocancionver,
			formatoarchivocancionver,
			notasversion
		FROM cancionversion
		WHERE codigocancionversion = $1
		  AND codigocancion = $2
		  AND fechahorabajaversion IS NULL
	`,
		codigoCancionVersion,
		codigoCancion,
	).Scan(
		&version.CodigoCancionVersion,
		&version.CodigoCancion,
		&version.NumeroVersion,
		&version.FechaHoraAltaVersion,
		&urlArchivo,
		&formatoArchivo,
		&notas,
	)

	if err != nil {
		return nil, err
	}

	if urlArchivo.Valid {
		version.URLArchivoCancionVer = &urlArchivo.String
	}

	if formatoArchivo.Valid {
		version.FormatoArchivoCancionVer = &formatoArchivo.String
	}

	if notas.Valid {
		version.NotasVersion = &notas.String
	}

	return &version, nil
}

func (r *cancionRepository) ListarPorProyecto(
	codigoProyecto int64,
) ([]dto.CancionListadoResponse, error) {

	rows, err := r.db.Query(`
		SELECT
			c.codigocancion,
			c.nombrecancion,
			cv.codigocancionversion,
			cv.numeroversion,
			cv.urlarchivocancionver,
			cv.formatoarchivocancionver
		FROM cancion c
		LEFT JOIN LATERAL (
			SELECT
				cvv.codigocancionversion,
				cvv.numeroversion,
				cvv.urlarchivocancionver,
				cvv.formatoarchivocancionver
			FROM cancionversion cvv
			WHERE cvv.codigocancion = c.codigocancion
			  AND cvv.fechahorabajaversion IS NULL
			ORDER BY cvv.numeroversion DESC
			LIMIT 1
		) cv ON TRUE
		WHERE c.codigoproyecto = $1
		  AND c.fechahorabajacancion IS NULL
		ORDER BY c.codigocancion DESC
	`,
		codigoProyecto,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	canciones := make(
		[]dto.CancionListadoResponse,
		0,
	)

	for rows.Next() {

		var cancion dto.CancionListadoResponse

		var codigoVersion sql.NullInt64
		var numeroVersion sql.NullInt64
		var urlArchivo sql.NullString
		var formatoArchivo sql.NullString

		err := rows.Scan(
			&cancion.CodigoCancion,
			&cancion.Nombre,
			&codigoVersion,
			&numeroVersion,
			&urlArchivo,
			&formatoArchivo,
		)

		if err != nil {
			return nil, err
		}

		if codigoVersion.Valid {

			version := dto.VersionActualCancionResponse{
				CodigoCancionVersion: codigoVersion.Int64,
				NumeroVersion:        int(numeroVersion.Int64),
				EtiquetaVersion: fmt.Sprintf(
					"v1.%d.0",
					numeroVersion.Int64-1,
				),
			}

			if urlArchivo.Valid {
				version.URLArchivo = &urlArchivo.String
			}

			if formatoArchivo.Valid {
				version.FormatoArchivo = &formatoArchivo.String
			}

			cancion.VersionActual = &version
		}

		canciones = append(
			canciones,
			cancion,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return canciones, nil
}

func (r *cancionRepository) ListarVersiones(
	codigoCancion int64,
) ([]dto.VersionCancionListadoResponse, error) {

	rows, err := r.db.Query(`
		SELECT
			codigocancionversion,
			numeroversion,
			fechahoraaltaversion,
			urlarchivocancionver,
			formatoarchivocancionver,
			notasversion
		FROM cancionversion
		WHERE codigocancion = $1
		  AND fechahorabajaversion IS NULL
		ORDER BY numeroversion DESC
	`,
		codigoCancion,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	versiones := make(
		[]dto.VersionCancionListadoResponse,
		0,
	)

	for rows.Next() {

		var version dto.VersionCancionListadoResponse

		var urlArchivo sql.NullString
		var formatoArchivo sql.NullString
		var notas sql.NullString

		err := rows.Scan(
			&version.CodigoCancionVersion,
			&version.NumeroVersion,
			&version.FechaHoraAlta,
			&urlArchivo,
			&formatoArchivo,
			&notas,
		)

		if err != nil {
			return nil, err
		}

		version.EtiquetaVersion = fmt.Sprintf(
			"v1.%d.0",
			version.NumeroVersion-1,
		)

		if urlArchivo.Valid {
			version.URLArchivo = &urlArchivo.String
		}

		if formatoArchivo.Valid {
			version.FormatoArchivo = &formatoArchivo.String
		}

		if notas.Valid {
			version.Notas = &notas.String
		}

		versiones = append(versiones, version)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return versiones, nil
}

// Las consultas de "mis canciones" se escriben completas y por separado (sin
// concatenar fragmentos) para que el análisis estático las reconozca como SQL
// constante. El FROM/WHERE debe mantenerse idéntico en ambas, porque
// totalItems tiene que reflejar el mismo subconjunto que se pagina;
// TestConsultasMisCancionesCompartenFiltro lo verifica.
//
// En ambas, $1 es el integrante y $2 el patrón de búsqueda ya escapado
// (cadena vacía = sin filtro por nombre).

const consultaContarMisCanciones = `
		SELECT COUNT(*)
		FROM integranteproyecto ip
		INNER JOIN proyecto p
			ON p.codigoproyecto = ip.codigoproyecto
		INNER JOIN cancion c
			ON c.codigoproyecto = p.codigoproyecto
		WHERE ip.codintegrante = $1
		  AND ip.fechahorabajaintegranteproy IS NULL
		  AND p.fechahorabajaproyecto IS NULL
		  AND c.fechahorabajacancion IS NULL
		  AND (
			$2::text = ''
			OR c.nombrecancion ILIKE '%' || $2::text || '%'
		  )
`

const consultaListarMisCanciones = `
		SELECT
			c.codigocancion,
			c.nombrecancion,
			p.codigoproyecto,
			p.nombreproyecto,
			cv.codigocancionversion,
			cv.numeroversion,
			cv.urlarchivocancionver,
			cv.formatoarchivocancionver
		FROM integranteproyecto ip
		INNER JOIN proyecto p
			ON p.codigoproyecto = ip.codigoproyecto
		INNER JOIN cancion c
			ON c.codigoproyecto = p.codigoproyecto
		LEFT JOIN LATERAL (
			SELECT
				cvv.codigocancionversion,
				cvv.numeroversion,
				cvv.urlarchivocancionver,
				cvv.formatoarchivocancionver,
				cvv.fechahoraaltaversion
			FROM cancionversion cvv
			WHERE cvv.codigocancion = c.codigocancion
			  AND cvv.fechahorabajaversion IS NULL
			ORDER BY cvv.numeroversion DESC
			LIMIT 1
		) cv ON TRUE
		WHERE ip.codintegrante = $1
		  AND ip.fechahorabajaintegranteproy IS NULL
		  AND p.fechahorabajaproyecto IS NULL
		  AND c.fechahorabajacancion IS NULL
		  AND (
			$2::text = ''
			OR c.nombrecancion ILIKE '%' || $2::text || '%'
		  )
		ORDER BY cv.fechahoraaltaversion DESC NULLS LAST,
			c.codigocancion DESC
		LIMIT $3 OFFSET $4
`

// escaparPatronLike neutraliza los comodines de LIKE (% y _) y el carácter
// de escape por defecto de PostgreSQL (\) para que la búsqueda sea literal.
func escaparPatronLike(texto string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(texto)
}

func (r *cancionRepository) ContarPorIntegrante(
	codigoIntegrante int64,
	busqueda string,
) (int, error) {

	var total int

	err := r.db.QueryRow(
		consultaContarMisCanciones,
		codigoIntegrante,
		escaparPatronLike(busqueda),
	).Scan(&total)

	if err != nil {
		return 0, err
	}

	return total, nil
}

// ListarPorIntegrante devuelve una página de las canciones de los proyectos
// en los que participa el integrante. Se ordena por la fecha de la versión
// actual (proxy de "última modificación"); las canciones sin versión quedan
// al final y codigocancion desempata para que la paginación sea estable.
func (r *cancionRepository) ListarPorIntegrante(
	codigoIntegrante int64,
	busqueda string,
	limite int,
	desplazamiento int,
) ([]dto.MiCancionListadoResponse, error) {

	rows, err := r.db.Query(
		consultaListarMisCanciones,
		codigoIntegrante,
		escaparPatronLike(busqueda),
		limite,
		desplazamiento,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	canciones := make(
		[]dto.MiCancionListadoResponse,
		0,
	)

	for rows.Next() {

		var cancion dto.MiCancionListadoResponse

		var codigoVersion sql.NullInt64
		var numeroVersion sql.NullInt64
		var urlArchivo sql.NullString
		var formatoArchivo sql.NullString

		err := rows.Scan(
			&cancion.CodigoCancion,
			&cancion.Nombre,
			&cancion.CodigoProyecto,
			&cancion.NombreProyecto,
			&codigoVersion,
			&numeroVersion,
			&urlArchivo,
			&formatoArchivo,
		)

		if err != nil {
			return nil, err
		}

		if codigoVersion.Valid {

			version := dto.VersionActualCancionResponse{
				CodigoCancionVersion: codigoVersion.Int64,
				NumeroVersion:        int(numeroVersion.Int64),
				EtiquetaVersion: fmt.Sprintf(
					"v1.%d.0",
					numeroVersion.Int64-1,
				),
			}

			if urlArchivo.Valid {
				version.URLArchivo = &urlArchivo.String
			}

			if formatoArchivo.Valid {
				version.FormatoArchivo = &formatoArchivo.String
			}

			cancion.VersionActual = &version
		}

		canciones = append(
			canciones,
			cancion,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return canciones, nil
}
