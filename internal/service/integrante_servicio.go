package service

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrPerfilIntegranteYaExiste = errors.New(
		"el usuario ya posee un perfil de integrante",
	)

	ErrNombreIntegranteObligatorio = errors.New(
		"el nombre del integrante es obligatorio",
	)

	ErrPerfilNoEncontrado = errors.New(
		"el usuario no posee un perfil",
	)
)

type IntegranteService interface {
	CrearPerfil(
		codigoUsuario int64,
		nombre string,
		descripcion *string,
	) (*model.Integrante, error)

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

func (s *integranteService) CrearPerfil(
	codigoUsuario int64,
	nombre string,
	descripcion *string,
) (*model.Integrante, error) {

	nombre = strings.TrimSpace(nombre)

	if nombre == "" {
		return nil, ErrNombreIntegranteObligatorio
	}

	perfilExistente, err :=
		s.repository.BuscarPorCodigoUsuario(codigoUsuario)

	if err == nil && perfilExistente != nil {
		return nil, ErrPerfilIntegranteYaExiste
	}

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if descripcion != nil {
		descripcionLimpia := strings.TrimSpace(*descripcion)
		descripcion = &descripcionLimpia
	}

	return s.repository.Crear(
		codigoUsuario,
		nombre,
		descripcion,
	)
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
