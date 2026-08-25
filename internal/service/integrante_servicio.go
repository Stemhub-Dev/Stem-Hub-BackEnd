package service

import (
	"database/sql"
	"errors"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrPerfilNoEncontrado = errors.New(
		"el usuario no posee un perfil",
	)
)

type IntegranteService interface {
	ObtenerPerfil(
		codigoUsuario int64,
	) (*model.Integrante, error)
}

type integranteService struct {
	repository repository.IntegranteRepository
}

func NewIntegranteService(
	repository repository.IntegranteRepository,
) IntegranteService {
	return &integranteService{
		repository: repository,
	}
}

func (s *integranteService) ObtenerPerfil(
	codigoUsuario int64,
) (*model.Integrante, error) {

	integrante, err :=
		s.repository.BuscarPorCodigoUsuario(codigoUsuario)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPerfilNoEncontrado
	}

	if err != nil {
		return nil, err
	}

	if integrante.FechaHoraBajaIntegrante != nil {
		return nil, ErrPerfilNoEncontrado
	}

	return integrante, nil
}
