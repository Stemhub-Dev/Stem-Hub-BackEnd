package repository

import (
	"database/sql"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type IntegranteRepository interface {
	BuscarPorCodigoUsuario(codigoUsuario int64) (*model.Integrante, error)
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

