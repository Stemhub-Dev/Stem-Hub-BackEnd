package service

import (
	"database/sql"
	"errors"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrUsuarioAdministracionNoEncontrado = errors.New("usuario no encontrado")
	ErrDatosAdministracionRequeridos     = errors.New("los campos esAdministradorSistema y activo son requeridos")
	ErrEstadoUsuarioRequerido            = errors.New("el campo activo es requerido")
)

type UsuarioAdministracionService interface {
	Listar() ([]dto.UsuarioAdministracionResponse, error)
	ObtenerPorID(codigoUsuario int64) (*dto.UsuarioAdministracionResponse, error)
	ObtenerRolesSistema(codigoUsuario int64) (*dto.UsuarioRolesResponse, error)
	ObtenerProyectosPorUsuario(codigoUsuario int64) (*dto.UsuarioProyectosResponse, error)
	ActualizarAdministracion(codigoUsuario int64, request dto.ActualizarAdministracionUsuarioRequest) (*dto.ActualizarAdministracionUsuarioResponse, error)
	CambiarEstado(codigoUsuario int64, request dto.CambiarEstadoUsuarioRequest) (*dto.CambiarEstadoUsuarioResponse, error)
}

type usuarioAdministracionService struct {
	repository repository.UsuarioAdministracionRepository
}

func NewUsuarioAdministracionService(
	repository repository.UsuarioAdministracionRepository,
) UsuarioAdministracionService {
	return &usuarioAdministracionService{
		repository: repository,
	}
}

func (s *usuarioAdministracionService) Listar() (
	[]dto.UsuarioAdministracionResponse,
	error,
) {
	return s.repository.Listar()
}

func (s *usuarioAdministracionService) ObtenerPorID(
	codigoUsuario int64,
) (*dto.UsuarioAdministracionResponse, error) {
	usuario, err := s.repository.ObtenerPorID(codigoUsuario)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUsuarioAdministracionNoEncontrado
		}

		return nil, err
	}

	return usuario, nil
}

func (s *usuarioAdministracionService) ObtenerRolesSistema(
	codigoUsuario int64,
) (*dto.UsuarioRolesResponse, error) {
	response, err := s.repository.ObtenerRolesSistema(codigoUsuario)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUsuarioAdministracionNoEncontrado
		}

		return nil, err
	}

	return response, nil
}

func (s *usuarioAdministracionService) ObtenerProyectosPorUsuario(
	codigoUsuario int64,
) (*dto.UsuarioProyectosResponse, error) {
	response, err := s.repository.ObtenerProyectosPorUsuario(codigoUsuario)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUsuarioAdministracionNoEncontrado
		}

		return nil, err
	}

	return response, nil
}

func (s *usuarioAdministracionService) ActualizarAdministracion(
	codigoUsuario int64,
	request dto.ActualizarAdministracionUsuarioRequest,
) (*dto.ActualizarAdministracionUsuarioResponse, error) {

	if request.EsAdministradorSistema == nil || request.Activo == nil {
		return nil, ErrDatosAdministracionRequeridos
	}

	response, err := s.repository.ActualizarAdministracion(
		codigoUsuario,
		*request.EsAdministradorSistema,
		*request.Activo,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUsuarioAdministracionNoEncontrado
		}

		return nil, err
	}

	return response, nil
}

func (s *usuarioAdministracionService) CambiarEstado(
	codigoUsuario int64,
	request dto.CambiarEstadoUsuarioRequest,
) (*dto.CambiarEstadoUsuarioResponse, error) {

	if request.Activo == nil {
		return nil, ErrEstadoUsuarioRequerido
	}

	response, err := s.repository.CambiarEstado(
		codigoUsuario,
		*request.Activo,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUsuarioAdministracionNoEncontrado
		}

		return nil, err
	}

	return response, nil
}
