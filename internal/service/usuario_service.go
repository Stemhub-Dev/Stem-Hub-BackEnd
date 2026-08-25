package service

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrUsuarioNoEncontrado         = errors.New("usuario no encontrado")
	ErrUsuarioInactivo             = errors.New("usuario inactivo")
	ErrNombreIntegranteObligatorio = errors.New("el nombre del integrante es obligatorio")
)

type UsuarioService interface {
	ObtenerUsuarioActivoPorIDAutenticacion(idAutenticacion string) (*model.Usuario, error)
	RegistrarUsuario(
		idAutenticacion string,
		email string,
		nombre string,
		descripcion *string,
	) (*model.Usuario, *model.Integrante, bool, error)
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
	nombre string,
	descripcion *string,
) (*model.Usuario, *model.Integrante, bool, error) {

	nombre = strings.TrimSpace(nombre)

	if nombre == "" {
		return nil, nil, false, ErrNombreIntegranteObligatorio
	}

	if descripcion != nil {
		descripcionLimpia := strings.TrimSpace(*descripcion)
		descripcion = &descripcionLimpia
	}

	usuario, integrante, creado, err := s.repository.Registrar(
		idAutenticacion,
		email,
		nombre,
		descripcion,
	)

	if err != nil {
		return nil, nil, false, err
	}

	if usuario.FechaHoraBajaUsuario != nil {
		return nil, nil, false, ErrUsuarioInactivo
	}

	return usuario, integrante, creado, nil
}
