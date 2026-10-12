package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

var ErrPermisosRolInvalidos = errors.New(
	"uno o más permisos son inválidos para el rol",
)

type RolPermisoRepository interface {
	ObtenerPermisosPorRol(
		codigoRol int64,
	) (*dto.RolPermisosResponse, error)

	ActualizarPermisosRol(
		codigoRol int64,
		codigosPermiso []int64,
	) (*dto.RolPermisosResponse, error)
}

type rolPermisoRepository struct {
	db *sql.DB
}

type queryer interface {
	QueryRow(
		query string,
		args ...any,
	) *sql.Row

	Query(
		query string,
		args ...any,
	) (*sql.Rows, error)
}

func NewRolPermisoRepository(
	db *sql.DB,
) RolPermisoRepository {

	return &rolPermisoRepository{
		db: db,
	}
}

func (r *rolPermisoRepository) ObtenerPermisosPorRol(
	codigoRol int64,
) (*dto.RolPermisosResponse, error) {

	return obtenerPermisosPorRolQuery(
		r.db,
		codigoRol,
	)
}

func (r *rolPermisoRepository) ActualizarPermisosRol(
	codigoRol int64,
	codigosPermiso []int64,
) (*dto.RolPermisosResponse, error) {

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf(
			"error al iniciar transacción: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var ambitoRol string

	err = tx.QueryRow(`
		SELECT ambitorol
		FROM rol
		WHERE codrol = $1
		  AND fechahorabajarol IS NULL
		FOR UPDATE
	`,
		codigoRol,
	).Scan(&ambitoRol)

	if err != nil {
		return nil, err
	}

	// Validamos que todos los permisos recibidos:
	// 1. existan,
	// 2. estén activos,
	// 3. pertenezcan al mismo ámbito que el rol.
	if len(codigosPermiso) > 0 {

		placeholders := make(
			[]string,
			len(codigosPermiso),
		)

		args := make(
			[]any,
			0,
			len(codigosPermiso)+1,
		)

		args = append(
			args,
			ambitoRol,
		)

		for i, codigoPermiso := range codigosPermiso {

			placeholders[i] =
				fmt.Sprintf(
					"$%d",
					i+2,
				)

			args = append(
				args,
				codigoPermiso,
			)
		}

		query := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM permiso
			WHERE ambitopermiso = $1
			  AND fechahorabajapermiso IS NULL
			  AND codigopermiso IN (%s)
		`,
			strings.Join(
				placeholders,
				",",
			),
		)

		var cantidadValidos int

		err = tx.QueryRow(
			query,
			args...,
		).Scan(&cantidadValidos)

		if err != nil {
			return nil, err
		}

		if cantidadValidos !=
			len(codigosPermiso) {

			return nil,
				ErrPermisosRolInvalidos
		}
	}

	// Damos de baja las asociaciones activas actuales
	// que ya no vienen en la nueva selección.
	if len(codigosPermiso) == 0 {

		_, err = tx.Exec(`
			UPDATE rolpermiso
			SET fechahorabajarolpermiso =
				CURRENT_TIMESTAMP
			WHERE codrol = $1
			  AND fechahorabajarolpermiso IS NULL
		`,
			codigoRol,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"error al quitar permisos del rol: %w",
				err,
			)
		}

	} else {

		placeholders := make(
			[]string,
			len(codigosPermiso),
		)

		args := make(
			[]any,
			0,
			len(codigosPermiso)+1,
		)

		args = append(
			args,
			codigoRol,
		)

		for i, codigoPermiso := range codigosPermiso {

			placeholders[i] =
				fmt.Sprintf(
					"$%d",
					i+2,
				)

			args = append(
				args,
				codigoPermiso,
			)
		}

		query := fmt.Sprintf(`
			UPDATE rolpermiso
			SET fechahorabajarolpermiso =
				CURRENT_TIMESTAMP
			WHERE codrol = $1
			  AND fechahorabajarolpermiso IS NULL
			  AND codigopermiso NOT IN (%s)
		`,
			strings.Join(
				placeholders,
				",",
			),
		)

		_, err = tx.Exec(
			query,
			args...,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"error al actualizar permisos del rol: %w",
				err,
			)
		}
	}

	// Insertamos únicamente las asociaciones solicitadas
	// que todavía no estén activas.
	for _, codigoPermiso := range codigosPermiso {

		_, err = tx.Exec(`
			INSERT INTO rolpermiso (
				codrol,
				codigopermiso,
				ambitorolpermiso
			)
			SELECT
				$1,
				$2,
				$3
			WHERE NOT EXISTS (
				SELECT 1
				FROM rolpermiso
				WHERE codrol = $1
				  AND codigopermiso = $2
				  AND fechahorabajarolpermiso IS NULL
			)
		`,
			codigoRol,
			codigoPermiso,
			ambitoRol,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"error al asociar permiso al rol: %w",
				err,
			)
		}
	}

	response, err :=
		obtenerPermisosPorRolQuery(
			tx,
			codigoRol,
		)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"error al confirmar permisos del rol: %w",
			err,
		)
	}

	return response, nil
}

func obtenerPermisosPorRolQuery(
	q queryer,
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

	response :=
		&dto.RolPermisosResponse{
			Permisos: make(
				[]dto.PermisoRolResponse,
				0,
			),
		}

	err := q.QueryRow(
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
		ORDER BY
			p.nombrepermiso,
			p.codigopermiso
	`

	rows, err := q.Query(
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

		response.Permisos =
			append(
				response.Permisos,
				permiso,
			)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return response, nil
}
