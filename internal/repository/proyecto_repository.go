package repository

import (
	"database/sql"
)

type ProyectoRepository interface {
	ExisteTipoProyectoActivo(codigoTipoProyecto int64) (bool, error)
	ObtenerAmbitoRolActivo(codRol int64) (string, error)
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
	) (int64, error)

	ExisteProyectoActivo(
		codigoProyecto int64,
	) (bool, error)

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
}

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

	if err := tx.Commit(); err != nil {
		return 0, err
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
			  AND (
					ip.espropietario = TRUE
					OR EXISTS (
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
			  AND (
					ip.espropietario = TRUE
					OR EXISTS (
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
