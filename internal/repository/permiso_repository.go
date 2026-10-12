package repository

import (
	"database/sql"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type PermisoRepository interface {
	TienePermiso(
		codigoUsuario int64,
		clavePermiso string,
	) (bool, error)

	Listar(
		incluirInactivos bool,
	) ([]model.Permiso, error)

	ObtenerPorID(
		codigoPermiso int64,
	) (*model.Permiso, error)

	ExistePorNombre(
		nombre string,
	) (bool, error)

	ExistePorClave(
		clave string,
	) (bool, error)

	ExistePorNombreExcluyendoID(
		nombre string,
		codigoPermiso int64,
	) (bool, error)

	ExistePorClaveExcluyendoID(
		clave string,
		codigoPermiso int64,
	) (bool, error)

	TieneAsignacionesActivas(
		codigoPermiso int64,
	) (bool, error)

	TieneAsignaciones(
		codigoPermiso int64,
	) (bool, error)

	Crear(
		nombre string,
		descripcion *string,
		ambito string,
		clave string,
	) (*model.Permiso, error)

	Editar(
		codigoPermiso int64,
		nombre string,
		descripcion *string,
		ambito string,
		clave string,
	) (*model.Permiso, error)

	CambiarEstado(
		codigoPermiso int64,
		activo bool,
	) (*model.Permiso, error)
}

type permisoRepository struct {
	db *sql.DB
}

func NewPermisoRepository(
	db *sql.DB,
) PermisoRepository {

	return &permisoRepository{
		db: db,
	}
}

func (r *permisoRepository) TienePermiso(
	codigoUsuario int64,
	clavePermiso string,
) (bool, error) {

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM usuariorol ur
			INNER JOIN rol r
				ON r.codrol = ur.codrol
			INNER JOIN rolpermiso rp
				ON rp.codrol = r.codrol
			INNER JOIN permiso p
				ON p.codigopermiso = rp.codigopermiso
			WHERE ur.codigousuario = $1
			  AND p.clavepermiso = $2

			  AND ur.fechahorabajausuariorol IS NULL
			  AND r.fechahorabajarol IS NULL
			  AND rp.fechahorabajarolpermiso IS NULL
			  AND p.fechahorabajapermiso IS NULL

			  AND ur.ambitorol = 'SISTEMA'
			  AND r.ambitorol = 'SISTEMA'
			  AND rp.ambitorolpermiso = 'SISTEMA'
			  AND p.ambitopermiso = 'SISTEMA'
		)
	`

	var tienePermiso bool

	err := r.db.QueryRow(
		query,
		codigoUsuario,
		clavePermiso,
	).Scan(&tienePermiso)

	if err != nil {
		return false, err
	}

	return tienePermiso, nil
}

func (r *permisoRepository) Listar(
	incluirInactivos bool,
) ([]model.Permiso, error) {

	query := `
		SELECT
			codigopermiso,
			nombrepermiso,
			descripcionpermiso,
			ambitopermiso,
			fechahorabajapermiso,
			clavepermiso
		FROM permiso
	`

	if !incluirInactivos {
		query += `
			WHERE fechahorabajapermiso IS NULL
		`
	}

	query += `
		ORDER BY
			ambitopermiso,
			nombrepermiso
	`

	rows, err := r.db.Query(query)

	if err != nil {
		return nil, fmt.Errorf(
			"error al consultar permisos: %w",
			err,
		)
	}

	defer rows.Close()

	permisos := make(
		[]model.Permiso,
		0,
	)

	for rows.Next() {

		var permiso model.Permiso

		err := rows.Scan(
			&permiso.CodigoPermiso,
			&permiso.NombrePermiso,
			&permiso.DescripcionPermiso,
			&permiso.AmbitoPermiso,
			&permiso.FechaHoraBajaPermiso,
			&permiso.ClavePermiso,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"error al leer permiso: %w",
				err,
			)
		}

		permisos = append(
			permisos,
			permiso,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error al recorrer permisos: %w",
			err,
		)
	}

	return permisos, nil
}

func (r *permisoRepository) ObtenerPorID(
	codigoPermiso int64,
) (*model.Permiso, error) {

	var permiso model.Permiso

	err := r.db.QueryRow(`
		SELECT
			codigopermiso,
			nombrepermiso,
			descripcionpermiso,
			ambitopermiso,
			fechahorabajapermiso,
			clavepermiso
		FROM permiso
		WHERE codigopermiso = $1
	`,
		codigoPermiso,
	).Scan(
		&permiso.CodigoPermiso,
		&permiso.NombrePermiso,
		&permiso.DescripcionPermiso,
		&permiso.AmbitoPermiso,
		&permiso.FechaHoraBajaPermiso,
		&permiso.ClavePermiso,
	)

	if err != nil {
		return nil, err
	}

	return &permiso, nil
}

func (r *permisoRepository) ExistePorNombre(
	nombre string,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM permiso
			WHERE LOWER(nombrepermiso) = LOWER($1)
		)
	`,
		nombre,
	).Scan(&existe)

	return existe, err
}

func (r *permisoRepository) ExistePorClave(
	clave string,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM permiso
			WHERE UPPER(clavepermiso) = UPPER($1)
		)
	`,
		clave,
	).Scan(&existe)

	return existe, err
}

func (r *permisoRepository) ExistePorNombreExcluyendoID(
	nombre string,
	codigoPermiso int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM permiso
			WHERE LOWER(nombrepermiso) = LOWER($1)
			  AND codigopermiso <> $2
		)
	`,
		nombre,
		codigoPermiso,
	).Scan(&existe)

	return existe, err
}

func (r *permisoRepository) ExistePorClaveExcluyendoID(
	clave string,
	codigoPermiso int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM permiso
			WHERE UPPER(clavepermiso) = UPPER($1)
			  AND codigopermiso <> $2
		)
	`,
		clave,
		codigoPermiso,
	).Scan(&existe)

	return existe, err
}

func (r *permisoRepository) TieneAsignacionesActivas(
	codigoPermiso int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM rolpermiso
			WHERE codigopermiso = $1
			  AND fechahorabajarolpermiso IS NULL
		)
	`,
		codigoPermiso,
	).Scan(&existe)

	return existe, err
}

func (r *permisoRepository) TieneAsignaciones(
	codigoPermiso int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM rolpermiso
			WHERE codigopermiso = $1
		)
	`,
		codigoPermiso,
	).Scan(&existe)

	return existe, err
}

func (r *permisoRepository) Crear(
	nombre string,
	descripcion *string,
	ambito string,
	clave string,
) (*model.Permiso, error) {

	var permiso model.Permiso

	err := r.db.QueryRow(`
		INSERT INTO permiso (
			nombrepermiso,
			descripcionpermiso,
			ambitopermiso,
			clavepermiso
		)
		VALUES (
			$1,
			$2,
			$3,
			$4
		)
		RETURNING
			codigopermiso,
			nombrepermiso,
			descripcionpermiso,
			ambitopermiso,
			fechahorabajapermiso,
			clavepermiso
	`,
		nombre,
		descripcion,
		ambito,
		clave,
	).Scan(
		&permiso.CodigoPermiso,
		&permiso.NombrePermiso,
		&permiso.DescripcionPermiso,
		&permiso.AmbitoPermiso,
		&permiso.FechaHoraBajaPermiso,
		&permiso.ClavePermiso,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"error al crear permiso: %w",
			err,
		)
	}

	return &permiso, nil
}

func (r *permisoRepository) Editar(
	codigoPermiso int64,
	nombre string,
	descripcion *string,
	ambito string,
	clave string,
) (*model.Permiso, error) {

	var permiso model.Permiso

	err := r.db.QueryRow(`
		UPDATE permiso
		SET
			nombrepermiso = $1,
			descripcionpermiso = $2,
			ambitopermiso = $3,
			clavepermiso = $4
		WHERE codigopermiso = $5
		RETURNING
			codigopermiso,
			nombrepermiso,
			descripcionpermiso,
			ambitopermiso,
			fechahorabajapermiso,
			clavepermiso
	`,
		nombre,
		descripcion,
		ambito,
		clave,
		codigoPermiso,
	).Scan(
		&permiso.CodigoPermiso,
		&permiso.NombrePermiso,
		&permiso.DescripcionPermiso,
		&permiso.AmbitoPermiso,
		&permiso.FechaHoraBajaPermiso,
		&permiso.ClavePermiso,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"error al editar permiso: %w",
			err,
		)
	}

	return &permiso, nil
}

func (r *permisoRepository) CambiarEstado(
	codigoPermiso int64,
	activo bool,
) (*model.Permiso, error) {

	var permiso model.Permiso

	err := r.db.QueryRow(`
		UPDATE permiso
		SET fechahorabajapermiso =
			CASE
				WHEN $1 THEN NULL
				ELSE CURRENT_TIMESTAMP
			END
		WHERE codigopermiso = $2
		RETURNING
			codigopermiso,
			nombrepermiso,
			descripcionpermiso,
			ambitopermiso,
			fechahorabajapermiso,
			clavepermiso
	`,
		activo,
		codigoPermiso,
	).Scan(
		&permiso.CodigoPermiso,
		&permiso.NombrePermiso,
		&permiso.DescripcionPermiso,
		&permiso.AmbitoPermiso,
		&permiso.FechaHoraBajaPermiso,
		&permiso.ClavePermiso,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"error al cambiar estado del permiso: %w",
			err,
		)
	}

	return &permiso, nil
}
