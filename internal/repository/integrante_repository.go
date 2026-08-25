package repository

import (
	"database/sql"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type IntegranteRepository interface {
	BuscarPorCodigoUsuario(codigoUsuario int64) (*model.Integrante, error)
	Crear(
		codigoUsuario int64,
		nombre string,
		descripcion *string,
	) (*model.Integrante, error)
}

type integranteRepository struct {
	db *sql.DB
}

func NewIntegranteRepository(db *sql.DB) IntegranteRepository {
	return &integranteRepository{
		db: db,
	}
}

func (r *integranteRepository) BuscarPorCodigoUsuario(
	codigoUsuario int64,
) (*model.Integrante, error) {

	query := `
		SELECT
			codintegrante,
			codigousuario,
			nombreintegrante,
			descripcionintegrante,
			fechahorabajaintegrante
		FROM integrante
		WHERE codigousuario = $1
	`

	var integrante model.Integrante

	err := r.db.QueryRow(
		query,
		codigoUsuario,
	).Scan(
		&integrante.CodIntegrante,
		&integrante.CodigoUsuario,
		&integrante.NombreIntegrante,
		&integrante.DescripcionIntegrante,
		&integrante.FechaHoraBajaIntegrante,
	)

	if err != nil {
		return nil, err
	}

	return &integrante, nil
}

func (r *integranteRepository) Crear(
	codigoUsuario int64,
	nombre string,
	descripcion *string,
) (*model.Integrante, error) {

	query := `
		INSERT INTO integrante (
			codigousuario,
			nombreintegrante,
			descripcionintegrante
		)
		VALUES ($1, $2, $3)
		RETURNING
			codintegrante,
			codigousuario,
			nombreintegrante,
			descripcionintegrante,
			fechahorabajaintegrante
	`

	var integrante model.Integrante

	err := r.db.QueryRow(
		query,
		codigoUsuario,
		nombre,
		descripcion,
	).Scan(
		&integrante.CodIntegrante,
		&integrante.CodigoUsuario,
		&integrante.NombreIntegrante,
		&integrante.DescripcionIntegrante,
		&integrante.FechaHoraBajaIntegrante,
	)

	if err != nil {
		return nil, err
	}

	return &integrante, nil
}
