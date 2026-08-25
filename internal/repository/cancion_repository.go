package repository

import "database/sql"

type CancionRepository interface {
	ExisteNombreEnProyecto(
		codigoProyecto int64,
		nombre string,
	) (bool, error)

	CrearConVersionInicial(
		codigoProyecto int64,
		nombre string,
		urlWAV *string,
		urlMP3 *string,
	) (int64, int64, error)

	ExisteCancionActivaEnProyecto(
		codigoProyecto int64,
		codigoCancion int64,
	) (bool, error)

	CrearVersion(
		codigoCancion int64,
		urlWAV *string,
		urlMP3 *string,
	) (int64, int, error)

	ExisteVersionActivaEnCancion(
		codigoCancion int64,
		codigoVersion int64,
	) (bool, error)
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

func (r *cancionRepository) CrearConVersionInicial(
	codigoProyecto int64,
	nombre string,
	urlWAV *string,
	urlMP3 *string,
) (int64, int64, error) {

	tx, err := r.db.Begin()

	if err != nil {
		return 0, 0, err
	}

	defer tx.Rollback()

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
		return 0, 0, err
	}

	var codigoCancionVersion int64

	err = tx.QueryRow(`
		INSERT INTO cancionversion (
			codigocancion,
			numeroversion,
			fechahoraaltaversion,
			urlversionwavcancionver,
			urlversionmp3cancionver
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
		urlWAV,
		urlMP3,
	).Scan(&codigoCancionVersion)

	if err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	return codigoCancion, codigoCancionVersion, nil
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

func (r *cancionRepository) CrearVersion(
	codigoCancion int64,
	urlWAV *string,
	urlMP3 *string,
) (int64, int, error) {

	tx, err := r.db.Begin()

	if err != nil {
		return 0, 0, err
	}

	defer tx.Rollback()

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
		return 0, 0, err
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
		return 0, 0, err
	}

	var codigoCancionVersion int64

	err = tx.QueryRow(`
		INSERT INTO cancionversion (
			codigocancion,
			numeroversion,
			fechahoraaltaversion,
			urlversionwavcancionver,
			urlversionmp3cancionver
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
		siguienteVersion,
		urlWAV,
		urlMP3,
	).Scan(&codigoCancionVersion)

	if err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}

	return codigoCancionVersion, siguienteVersion, nil
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
