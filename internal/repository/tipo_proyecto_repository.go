package repository

import (
	"database/sql"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type TipoProyectoRepository struct {
	db *sql.DB
}

func NewTipoProyectoRepository(db *sql.DB) *TipoProyectoRepository {
	return &TipoProyectoRepository{db: db}
}

func (r *TipoProyectoRepository) Listar() ([]model.TipoProyecto, error) {

	query := `
		SELECT
			codtipoproy,
			nombretipoproy,
			descripciontipoproy,
			fechahorabajatipoproy
		FROM tipoproyecto
		WHERE fechahorabajatipoproy IS NULL
		ORDER BY nombretipoproy
	`

	rows, err := r.db.Query(query)

	if err != nil {
		return nil, fmt.Errorf("error al consultar tipos de proyecto: %w", err)
	}
	defer rows.Close()

	tipos := make([]model.TipoProyecto, 0)

	for rows.Next() {
		var tipo model.TipoProyecto

		err := rows.Scan(
			&tipo.CodTipoProy,
			&tipo.NombreTipoProy,
			&tipo.DescripcionTipoProy,
			&tipo.FechaHoraBajaTipoProy,
		)

		if err != nil {
			return nil, fmt.Errorf("error al leer tipo de proyecto: %w", err)
		}

		tipos = append(tipos, tipo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error al recorrer tipos de proyecto: %w", err)
	}

	return tipos, nil
}
