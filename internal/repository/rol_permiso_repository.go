package repository

import (
	"database/sql"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

type RolPermisoRepository interface {
	ObtenerPermisosPorRol(codigoRol int64) (*dto.RolPermisosResponse, error)
}

type rolPermisoRepository struct {
	db *sql.DB
}

func NewRolPermisoRepository(db *sql.DB) RolPermisoRepository {
	return &rolPermisoRepository{
		db: db,
	}
}

func (r *rolPermisoRepository) ObtenerPermisosPorRol(
	codigoRol int64,
) (*dto.RolPermisosResponse, error) {
	const queryRol = `
		SELECT
			codrol,
			nombrerol,
			ambitorol
		FROM rol
		WHERE codrol = $1
		  AND fechahorabajarol IS NULL
	`

	response := &dto.RolPermisosResponse{
		Permisos: make([]dto.PermisoRolResponse, 0),
	}

	err := r.db.QueryRow(
		queryRol,
		codigoRol,
	).Scan(
		&response.CodigoRol,
		&response.NombreRol,
		&response.AmbitoRol,
	)
	if err != nil {
		return nil, err
	}

	const queryPermisos = `
		SELECT
			p.codigopermiso,
			p.clavepermiso,
			p.nombrepermiso,
			p.descripcionpermiso,
			EXISTS (
				SELECT 1
				FROM rolpermiso rp
				WHERE rp.codrol = $1
				  AND rp.codigopermiso = p.codigopermiso
				  AND rp.ambitorolpermiso = $2
				  AND rp.fechahorabajarolpermiso IS NULL
			) AS asignado
		FROM permiso p
		WHERE p.ambitopermiso = $2
		  AND p.fechahorabajapermiso IS NULL
		ORDER BY p.nombrepermiso, p.codigopermiso
	`

	rows, err := r.db.Query(
		queryPermisos,
		codigoRol,
		response.AmbitoRol,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var permiso dto.PermisoRolResponse

		err = rows.Scan(
			&permiso.CodigoPermiso,
			&permiso.ClavePermiso,
			&permiso.NombrePermiso,
			&permiso.DescripcionPermiso,
			&permiso.Asignado,
		)
		if err != nil {
			return nil, err
		}

		response.Permisos = append(
			response.Permisos,
			permiso,
		)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return response, nil
}
