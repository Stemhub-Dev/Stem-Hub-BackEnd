package service

import (
	"errors"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrUsuarioRolUsuarioNoEncontrado = errors.New("el usuario no existe")
	ErrUsuarioRolRolNoEncontrado     = errors.New("el rol no existe")
	ErrUsuarioRolYaAsignado          = errors.New("el usuario ya posee el rol")
)

type UsuarioRolService interface {
	AsignarRol(codigoUsuario int64, codRol int64) error
}

type usuarioRolService struct {
	repository repository.UsuarioRolRepository
}

func NewUsuarioRolService(
	repository repository.UsuarioRolRepository,
) UsuarioRolService {
	return &usuarioRolService{
		repository: repository,
	}
}

func (s *usuarioRolService) AsignarRol(
	codigoUsuario int64,
	codRol int64,
) error {

	existeUsuario, err := s.repository.ExisteUsuarioActivo(
		codigoUsuario,
	)

	if err != nil {
		return err
	}

	if !existeUsuario {
		return ErrUsuarioRolUsuarioNoEncontrado
	}

	existeRol, err := s.repository.ExisteRolActivo(
		codRol,
	)

	if err != nil {
		return err
	}

	if !existeRol {
		return ErrUsuarioRolRolNoEncontrado
	}

	yaAsignado, err := s.repository.TieneRolActivo(
		codigoUsuario,
		codRol,
	)

	if err != nil {
		return err
	}

	if yaAsignado {
		return ErrUsuarioRolYaAsignado
	}

	return s.repository.AsignarRol(
		codigoUsuario,
		codRol,
	)
}
