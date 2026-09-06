package repository

import (
	"database/sql"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type PermisoRepository interface {
	TienePermiso(codigoUsuario int64, clavePermiso string) (bool, error)
	ListarActivos() ([]model.Permiso, error)
}

type permisoRepository struct {
	db *sql.DB
}

func NewPermisoRepository(db *sql.DB) PermisoRepository {
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

func (r *permisoRepository) ListarActivos() (
	[]model.Permiso,
	error,
) {

	rows, err := r.db.Query(`
		SELECT
			codigopermiso,
			nombrepermiso,
			descripcionpermiso,
			ambitopermiso,
			fechahorabajapermiso,
			clavepermiso
		FROM permiso
		WHERE fechahorabajapermiso IS NULL
		ORDER BY
			ambitopermiso,
			nombrepermiso
	`)

	if err != nil {
		return nil, err
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
			return nil, err
		}

		permisos = append(
			permisos,
			permiso,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return permisos, nil
}
