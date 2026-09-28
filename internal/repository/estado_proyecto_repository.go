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

func (r *EstadoProyectoRepository) Listar() (
	[]model.EstadoProyecto,
	error,
) {

	query := `
		SELECT
			codestadoproy,
			nombreestadoproy,
			descripcionestadoproy,
			fechahorabajaestadoproy
		FROM estadoproyecto
		WHERE fechahorabajaestadoproy IS NULL
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
