package repository

import (
	"database/sql"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type PreguntaFrecuenteRepository interface {
	ListarActivas() ([]model.PreguntaFrecuente, error)
}

type preguntaFrecuenteRepository struct {
	db *sql.DB
}

func NewPreguntaFrecuenteRepository(db *sql.DB) PreguntaFrecuenteRepository {
	return &preguntaFrecuenteRepository{db: db}
}

// ListarActivas respeta el orden de carga: la tabla no tiene un campo de
// orden y es el que define cómo se presentan las funcionalidades.
func (r *preguntaFrecuenteRepository) ListarActivas() ([]model.PreguntaFrecuente, error) {
	rows, err := r.db.Query(`
		SELECT
			codigopreguntafrecuente,
			funcionalidadprincipal,
			preguntafrecuente,
			respuestafrecuente
		FROM preguntafrecuente
		WHERE fechahorabajapreguntafrecuente IS NULL
		ORDER BY codigopreguntafrecuente
	`)
	if err != nil {
		return nil, fmt.Errorf("error al consultar preguntas frecuentes: %w", err)
	}
	defer rows.Close()

	preguntas := make([]model.PreguntaFrecuente, 0)

	for rows.Next() {
		var pregunta model.PreguntaFrecuente

		if err := rows.Scan(
			&pregunta.CodigoPreguntaFrecuente,
			&pregunta.FuncionalidadPrincipal,
			&pregunta.PreguntaFrecuente,
			&pregunta.RespuestaFrecuente,
		); err != nil {
			return nil, fmt.Errorf("error al leer pregunta frecuente: %w", err)
		}

		preguntas = append(preguntas, pregunta)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error al recorrer preguntas frecuentes: %w", err)
	}

	return preguntas, nil
}
