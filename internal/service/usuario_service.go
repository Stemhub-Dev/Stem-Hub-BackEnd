package service

import (
	"database/sql"
	"errors"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrUsuarioNoEncontrado = errors.New("usuario no encontrado")
	ErrUsuarioInactivo     = errors.New("usuario inactivo")
)

type UsuarioService interface {
	ObtenerUsuarioActivoPorIDAutenticacion(idAutenticacion string) (*model.Usuario, error)
	RegistrarUsuario(
		idAutenticacion string,
		email string,
	) (*model.Usuario, bool, error)
}

type usuarioService struct {
	repository repository.UsuarioRepository
}

func NewUsuarioService(repository repository.UsuarioRepository) *usuarioService {
	return &usuarioService{
		repository: repository,
	}
}

func (s *usuarioService) ObtenerUsuarioActivoPorIDAutenticacion(idAutenticacion string) (*model.Usuario, error) {
	usuario, err := s.repository.BuscarPorIDAutenticacion(idAutenticacion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUsuarioNoEncontrado
		}

		return nil, err
	}

	if usuario.FechaHoraBajaUsuario != nil {
		return nil, ErrUsuarioInactivo
	}
	return usuario, nil
}

func (s *usuarioService) RegistrarUsuario(
	idAutenticacion string,
	email string,
) (*model.Usuario, bool, error) {

	usuario, creado, err := s.repository.Registrar(
		idAutenticacion,
		email,
	)

	if err != nil {
		return nil, false, err
	}

	if usuario.FechaHoraBajaUsuario != nil {
		return nil, false, ErrUsuarioInactivo
	}

	return usuario, creado, nil
}
