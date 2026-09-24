package repository

import (
	"database/sql"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type PreguntaFrecuenteRepository struct {
	db *sql.DB
}

func NewPreguntaFrecuenteRepository(db *sql.DB) *PreguntaFrecuenteRepository {
	return &PreguntaFrecuenteRepository{db: db}
}

func (r *PreguntaFrecuenteRepository) Listar() ([]model.PreguntaFrecuente, error) {

	query := `
	SELECT codigopreguntafrecuente,
	funcionalidadprincipal,
	preguntafrecuente,
	respuestafrecuente,
	fechahorabajapreguntafrecuente
	FROM preguntafrecuente
	WHERE fechahorabajapreguntafrecuente IS NULL
	ORDER BY codigopreguntafrecuente
	`

	rows, err := r.db.Query(query)

	if err != nil {
		return nil, fmt.Errorf("error al consultar preguntas frecuentes: %w", err)
	}
	defer rows.Close()

	preguntas := make([]model.PreguntaFrecuente, 0)

	for rows.Next() {
		var pregunta model.PreguntaFrecuente

		err := rows.Scan(
			&pregunta.CodigoPreguntaFrecuente,
			&pregunta.FuncionalidadPrincipal,
			&pregunta.PreguntaFrecuente,
			&pregunta.RespuestaFrecuente,
			&pregunta.FechaHoraBajaPreguntaFrecuente,
		)

		if err != nil {
			return nil, fmt.Errorf("error al leer pregunta frecuente: %w", err)
		}

		preguntas = append(preguntas, pregunta)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error al recorrer preguntas frecuentes: %w", err)
	}

	return preguntas, nil

}

func (r *PreguntaFrecuenteRepository) BuscarPorID(id int64) (*model.PreguntaFrecuente, error) {
	query := `
	SELECT codigopreguntafrecuente,
		funcionalidadprincipal,
		preguntafrecuente,
		respuestafrecuente,
		fechahorabajapreguntafrecuente
	FROM preguntafrecuente
	WHERE codigopreguntafrecuente = $1
		AND fechahorabajapreguntafrecuente IS NULL
	`

	var pregunta model.PreguntaFrecuente

	err := r.db.QueryRow(query, id).Scan(
		&pregunta.CodigoPreguntaFrecuente,
		&pregunta.FuncionalidadPrincipal,
		&pregunta.PreguntaFrecuente,
		&pregunta.RespuestaFrecuente,
		&pregunta.FechaHoraBajaPreguntaFrecuente,
	)

	if err != nil {
		return nil, err
	}

	return &pregunta, nil
}
