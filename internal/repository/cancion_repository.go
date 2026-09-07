package repository

import (
	"database/sql"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
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

	FinalizarCreacionVersion(
		tx *sql.Tx,
		codigoCancion int64,
		numeroVersion int,
		urlArchivo string,
		formatoArchivo string,
	) (int64, error)

	ExisteVersionActivaEnCancion(
		codigoCancion int64,
		codigoVersion int64,
	) (bool, error)

	ListarPorProyecto(
		codigoProyecto int64,
	) ([]dto.CancionListadoResponse, error)

	ListarVersiones(
		codigoCancion int64,
	) ([]dto.VersionCancionListadoResponse, error)
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

func (r *cancionRepository) FinalizarCreacionVersion(
	tx *sql.Tx,
	codigoCancion int64,
	numeroVersion int,
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
			$2,
			CURRENT_TIMESTAMP,
			$3,
			$4
		)
		RETURNING codigocancionversion
	`,
		codigoCancion,
		numeroVersion,
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
			formatoarchivocancionver
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

		err := rows.Scan(
			&version.CodigoCancionVersion,
			&version.NumeroVersion,
			&version.FechaHoraAlta,
			&urlArchivo,
			&formatoArchivo,
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

		versiones = append(versiones, version)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return versiones, nil
}
