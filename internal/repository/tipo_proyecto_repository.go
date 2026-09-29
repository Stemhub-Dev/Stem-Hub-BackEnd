package repository

import (
	"database/sql"
	"fmt"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type TipoProyectoRepository struct {
	db *sql.DB
}

func NewTipoProyectoRepository(
	db *sql.DB,
) *TipoProyectoRepository {

	return &TipoProyectoRepository{
		db: db,
	}
}

func (r *TipoProyectoRepository) Listar(
	incluirInactivos bool,
) ([]model.TipoProyecto, error) {

	query := `
		SELECT
			codtipoproy,
			nombretipoproy,
			descripciontipoproy,
			fechahorabajatipoproy
		FROM tipoproyecto
	`

	if !incluirInactivos {
		query += `
			WHERE fechahorabajatipoproy IS NULL
		`
	}

	query += `
		ORDER BY nombretipoproy
	`

	rows, err := r.db.Query(query)

	if err != nil {
		return nil, fmt.Errorf(
			"error al consultar tipos de proyecto: %w",
			err,
		)
	}

	defer rows.Close()

	tipos := make(
		[]model.TipoProyecto,
		0,
	)

	for rows.Next() {

		var tipo model.TipoProyecto

		err := rows.Scan(
			&tipo.CodTipoProy,
			&tipo.NombreTipoProy,
			&tipo.DescripcionTipoProy,
			&tipo.FechaHoraBajaTipoProy,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"error al leer tipo de proyecto: %w",
				err,
			)
		}

		tipos = append(
			tipos,
			tipo,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"error al recorrer tipos de proyecto: %w",
			err,
		)
	}

	return tipos, nil
}

func (r *TipoProyectoRepository) ExistePorNombre(
	nombre string,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM tipoproyecto
			WHERE LOWER(nombretipoproy) = LOWER($1)
		)
	`,
		nombre,
	).Scan(
		&existe,
	)

	if err != nil {
		return false, fmt.Errorf(
			"error al verificar tipo de proyecto existente: %w",
			err,
		)
	}

	return existe, nil
}

func (r *TipoProyectoRepository) ExistePorID(
	id int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM tipoproyecto
			WHERE codtipoproy = $1
		)
	`,
		id,
	).Scan(
		&existe,
	)

	if err != nil {
		return false, fmt.Errorf(
			"error al verificar tipo de proyecto por ID: %w",
			err,
		)
	}

	return existe, nil
}

func (r *TipoProyectoRepository) ExistePorNombreExcluyendoID(
	nombre string,
	id int64,
) (bool, error) {

	var existe bool

	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM tipoproyecto
			WHERE LOWER(nombretipoproy) = LOWER($1)
			  AND codtipoproy <> $2
		)
	`,
		nombre,
		id,
	).Scan(
		&existe,
	)

	if err != nil {
		return false, fmt.Errorf(
			"error al verificar nombre de tipo de proyecto: %w",
			err,
		)
	}

	return existe, nil
}

func (r *TipoProyectoRepository) Crear(
	nombre string,
) (model.TipoProyecto, error) {

	var tipo model.TipoProyecto

	err := r.db.QueryRow(`
		INSERT INTO tipoproyecto (
			nombretipoproy
		)
		VALUES ($1)
		RETURNING
			codtipoproy,
			nombretipoproy,
			descripciontipoproy,
			fechahorabajatipoproy
	`,
		nombre,
	).Scan(
		&tipo.CodTipoProy,
		&tipo.NombreTipoProy,
		&tipo.DescripcionTipoProy,
		&tipo.FechaHoraBajaTipoProy,
	)

	if err != nil {
		return model.TipoProyecto{},
			fmt.Errorf(
				"error al crear tipo de proyecto: %w",
				err,
			)
	}

	return tipo, nil
}

func (r *TipoProyectoRepository) Editar(
	id int64,
	nombre string,
) (model.TipoProyecto, error) {

	var tipo model.TipoProyecto

	err := r.db.QueryRow(`
		UPDATE tipoproyecto
		SET nombretipoproy = $1
		WHERE codtipoproy = $2
		RETURNING
			codtipoproy,
			nombretipoproy,
			descripciontipoproy,
			fechahorabajatipoproy
	`,
		nombre,
		id,
	).Scan(
		&tipo.CodTipoProy,
		&tipo.NombreTipoProy,
		&tipo.DescripcionTipoProy,
		&tipo.FechaHoraBajaTipoProy,
	)

	if err != nil {
		return model.TipoProyecto{},
			fmt.Errorf(
				"error al editar tipo de proyecto: %w",
				err,
			)
	}

	return tipo, nil
}

func (r *TipoProyectoRepository) CambiarEstado(
	id int64,
	activo bool,
) (model.TipoProyecto, error) {

	var tipo model.TipoProyecto

	err := r.db.QueryRow(`
		UPDATE tipoproyecto
		SET fechahorabajatipoproy =
			CASE
				WHEN $1 THEN NULL
				ELSE CURRENT_TIMESTAMP
			END
		WHERE codtipoproy = $2
		RETURNING
			codtipoproy,
			nombretipoproy,
			descripciontipoproy,
			fechahorabajatipoproy
	`,
		activo,
		id,
	).Scan(
		&tipo.CodTipoProy,
		&tipo.NombreTipoProy,
		&tipo.DescripcionTipoProy,
		&tipo.FechaHoraBajaTipoProy,
	)

	if err != nil {
		return model.TipoProyecto{},
			fmt.Errorf(
				"error al cambiar estado del tipo de proyecto: %w",
				err,
			)
	}

	return tipo, nil
}
