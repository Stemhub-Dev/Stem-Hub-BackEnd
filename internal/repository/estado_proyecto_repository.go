package repository

import (
	"database/sql"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type EstadoProyectoRepository struct {
	db *sql.DB
}

func NewEstadoProyectoRepository(
	db *sql.DB,
) *EstadoProyectoRepository {

	return &EstadoProyectoRepository{
		db: db,
	}
}

func (r *EstadoProyectoRepository) Listar(
	incluirInactivos bool,
) ([]model.EstadoProyecto, error) {

	query := `
		SELECT
			codestadoproy,
			nombreestadoproy,
			descripcionestadoproy,
			fechahorabajaestadoproy
		FROM estadoproyecto
	`

	if !incluirInactivos {
		query += `
			WHERE fechahorabajaestadoproy IS NULL
		`
	}

	query += `
		ORDER BY nombreestadoproy
	`

	rows, err := r.db.Query(query)

	if err != nil {
		return nil, fmt.Errorf(
			"error al consultar estados de proyecto: %w",
			err,
		)
	}

	defer rows.Close()

	estados := make(
		[]model.EstadoProyecto,
		0,
	)

	for rows.Next() {

		var estado model.EstadoProyecto

		err := rows.Scan(
			&estado.CodEstadoProy,
			&estado.NombreEstadoProy,
			&estado.DescripcionEstadoProy,
			&estado.FechaHoraBajaEstadoProy,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"error al leer estado de proyecto: %w",
				err,
			)
		}

		estados = append(
			estados,
			estado,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error al recorrer estados de proyecto: %w",
			err,
		)
	}

	return estados, nil
}

func (r *EstadoProyectoRepository) ExistePorNombre(
	nombre string,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM estadoproyecto
			WHERE LOWER(nombreestadoproy) = LOWER($1)
		)
	`, nombre).Scan(&existe)

	if err != nil {
		return false, fmt.Errorf(
			"error al verificar estado de proyecto existente: %w",
			err,
		)
	}

	return existe, nil
}

func (r *EstadoProyectoRepository) ExistePorID(
	id int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM estadoproyecto
			WHERE codestadoproy = $1
		)
	`, id).Scan(&existe)

	if err != nil {
		return false, fmt.Errorf(
			"error al verificar estado de proyecto por ID: %w",
			err,
		)
	}

	return existe, nil
}

func (r *EstadoProyectoRepository) ExistePorNombreExcluyendoID(
	nombre string,
	id int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM estadoproyecto
			WHERE LOWER(nombreestadoproy) = LOWER($1)
			  AND codestadoproy <> $2
		)
	`, nombre, id).Scan(&existe)

	if err != nil {
		return false, fmt.Errorf(
			"error al verificar nombre de estado de proyecto: %w",
			err,
		)
	}

	return existe, nil
}

func (r *EstadoProyectoRepository) Crear(
	nombre string,
) (model.EstadoProyecto, error) {

	var estado model.EstadoProyecto

	err := r.db.QueryRow(`
		INSERT INTO estadoproyecto (
			nombreestadoproy
		)
		VALUES ($1)
		RETURNING
			codestadoproy,
			nombreestadoproy,
			descripcionestadoproy,
			fechahorabajaestadoproy
	`, nombre).Scan(
		&estado.CodEstadoProy,
		&estado.NombreEstadoProy,
		&estado.DescripcionEstadoProy,
		&estado.FechaHoraBajaEstadoProy,
	)

	if err != nil {
		return model.EstadoProyecto{},
			fmt.Errorf(
				"error al crear estado de proyecto: %w",
				err,
			)
	}

	return estado, nil
}

func (r *EstadoProyectoRepository) Editar(
	id int64,
	nombre string,
) (model.EstadoProyecto, error) {

	var estado model.EstadoProyecto

	err := r.db.QueryRow(`
		UPDATE estadoproyecto
		SET nombreestadoproy = $1
		WHERE codestadoproy = $2
		RETURNING
			codestadoproy,
			nombreestadoproy,
			descripcionestadoproy,
			fechahorabajaestadoproy
	`,
		nombre,
		id,
	).Scan(
		&estado.CodEstadoProy,
		&estado.NombreEstadoProy,
		&estado.DescripcionEstadoProy,
		&estado.FechaHoraBajaEstadoProy,
	)

	if err != nil {
		return model.EstadoProyecto{},
			fmt.Errorf(
				"error al editar estado de proyecto: %w",
				err,
			)
	}

	return estado, nil
}

func (r *EstadoProyectoRepository) CambiarEstado(
	id int64,
	activo bool,
) (model.EstadoProyecto, error) {

	var estado model.EstadoProyecto

	err := r.db.QueryRow(`
		UPDATE estadoproyecto
		SET fechahorabajaestadoproy =
			CASE
				WHEN $1 THEN NULL
				ELSE CURRENT_TIMESTAMP
			END
		WHERE codestadoproy = $2
		RETURNING
			codestadoproy,
			nombreestadoproy,
			descripcionestadoproy,
			fechahorabajaestadoproy
	`,
		activo,
		id,
	).Scan(
		&estado.CodEstadoProy,
		&estado.NombreEstadoProy,
		&estado.DescripcionEstadoProy,
		&estado.FechaHoraBajaEstadoProy,
	)

	if err != nil {
		return model.EstadoProyecto{},
			fmt.Errorf(
				"error al cambiar estado del estado de proyecto: %w",
				err,
			)
	}

	return estado, nil
}
