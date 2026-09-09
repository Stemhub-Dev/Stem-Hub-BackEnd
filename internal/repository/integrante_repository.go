package repository

import (
	"database/sql"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type IntegranteRepository interface {
	BuscarPorCodigoUsuario(codigoUsuario int64) (*model.Integrante, error)

	ActualizarPerfil(
		codigoIntegrante int64,
		nombre string,
		descripcion *string,
		avatarObjectKey *string,
	) error
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
			urlavatarintegrante,
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
		&integrante.AvatarObjectKey,
		&integrante.FechaHoraBajaIntegrante,
	)

	if err != nil {
		return nil, err
	}

	return &integrante, nil
}

func (r *integranteRepository) ActualizarPerfil(
	codigoIntegrante int64,
	nombre string,
	descripcion *string,
	avatarObjectKey *string,
) error {

	_, err := r.db.Exec(
		`
			UPDATE integrante
			SET
				nombreintegrante = $1,
				descripcionintegrante = $2,
				urlavatarintegrante = $3
			WHERE codintegrante = $4
		`,
		nombre,
		descripcion,
		avatarObjectKey,
		codigoIntegrante,
	)

	return err
}

