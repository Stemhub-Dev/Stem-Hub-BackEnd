package repository

import (
	"database/sql"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type GeneroMusicalRepository struct {
	db *sql.DB
}

func NewGeneroMusicalRepository(db *sql.DB) *GeneroMusicalRepository {
	return &GeneroMusicalRepository{db: db}
}

func (r *GeneroMusicalRepository) Listar() ([]model.GeneroMusicalProyecto, error) {

	query := `
	SELECT codigogeneroproy,
	nombregeneroproy,
	descripciongeneroproy,
	fechahorabajageneroproy
	FROM generomusicalproyecto
	ORDER BY nombregeneroproy
	`

	rows, err := r.db.Query(query)

	if err != nil {
		return nil, fmt.Errorf("error al consultar géneros musicales: %W", err)
	}
	defer rows.Close()

	generos := make([]model.GeneroMusicalProyecto, 0)

	for rows.Next() {
		var genero model.GeneroMusicalProyecto

		err := rows.Scan(
			&genero.CodigoGeneroProy,
			&genero.NombreGeneroProy,
			&genero.DescripcionGeneroProy,
			&genero.FechaHoraBajaGeneroProy,
		)

		if err != nil {
			return nil, fmt.Errorf("Error al leer rol: %W", err)
		}

		generos = append(generos, genero)

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("Error al recorrer roles: %W", err)
		}
	}

	return generos, nil

}

func (r *GeneroMusicalRepository) ExistePorNombre(nombre string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM generomusicalproyecto
			WHERE LOWER(nombregeneroproy) = LOWER($1)
		)
	`

	var existe bool

	err := r.db.QueryRow(query, nombre).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("error al verificar género musical existente: %W", err)
	}

	return existe, nil
}

func (r *GeneroMusicalRepository) Crear(
	nombre string) (model.GeneroMusicalProyecto, error) {
	query := `
		INSERT INTO generomusicalproyecto (
			nombregeneroproy
		)
		VALUES ($1)
		RETURNING
			codigogeneroproy,
			nombregeneroproy,
			descripciongeneroproy,
			fechahorabajageneroproy
	`

	var genero model.GeneroMusicalProyecto

	err := r.db.QueryRow(query, nombre).Scan(
		&genero.CodigoGeneroProy,
		&genero.NombreGeneroProy,
		&genero.DescripcionGeneroProy,
		&genero.FechaHoraBajaGeneroProy,
	)

	if err != nil {
		return model.GeneroMusicalProyecto{}, fmt.Errorf("Error al crear género musical: %W", err)
	}

	return genero, nil
}

func (r *GeneroMusicalRepository) ExistePorID(
	id int64) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM generomusicalproyecto
			WHERE codigogeneroproy = $1
		)
	`

	var existe bool

	err := r.db.QueryRow(query, id).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("error al verificar género musical por ID: %w", err)
	}

	return existe, nil
}

func (r *GeneroMusicalRepository) ExistePorNombreExcluyendoID(
	nombre string,
	id int64) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM generomusicalproyecto
			WHERE LOWER(nombregeneroproy) = LOWER($1)
			  AND codigogeneroproy <> $2
		)
	`

	var existe bool

	err := r.db.QueryRow(query, nombre, id).Scan(&existe)
	if err != nil {
		return false, fmt.Errorf("Error al verificar nombre de género musical: %w", err)
	}

	return existe, nil
}

func (r *GeneroMusicalRepository) Editar(
	id int64,
	nombre string) (model.GeneroMusicalProyecto, error) {
	query := `
		UPDATE generomusicalproyecto
		SET nombregeneroproy = $1
		WHERE codigogeneroproy = $2
		RETURNING
			codigogeneroproy,
			nombregeneroproy,
			descripciongeneroproy,
			fechahorabajageneroproy
	`

	var genero model.GeneroMusicalProyecto

	err := r.db.QueryRow(query, nombre, id).Scan(
		&genero.CodigoGeneroProy,
		&genero.NombreGeneroProy,
		&genero.DescripcionGeneroProy,
		&genero.FechaHoraBajaGeneroProy,
	)
	if err != nil {
		return model.GeneroMusicalProyecto{}, fmt.Errorf("Error al editar género musical: %w", err)
	}

	return genero, nil
}

func (r *GeneroMusicalRepository) CambiarEstado(
	id int64,
	activo bool,
) (model.GeneroMusicalProyecto, error) {

	query := `
		UPDATE generomusicalproyecto
		SET fechahorabajageneroproy =
			CASE
				WHEN $1 THEN NULL
				ELSE CURRENT_TIMESTAMP
			END
		WHERE codigogeneroproy = $2
		RETURNING
			codigogeneroproy,
			nombregeneroproy,
			descripciongeneroproy,
			fechahorabajageneroproy
	`

	var genero model.GeneroMusicalProyecto

	err := r.db.QueryRow(
		query,
		activo,
		id,
	).Scan(
		&genero.CodigoGeneroProy,
		&genero.NombreGeneroProy,
		&genero.DescripcionGeneroProy,
		&genero.FechaHoraBajaGeneroProy,
	)

	if err != nil {
		return model.GeneroMusicalProyecto{},
			fmt.Errorf("Error al cambiar estado del género musical: %w", err)
	}

	return genero, nil
}
