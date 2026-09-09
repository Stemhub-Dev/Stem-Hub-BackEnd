package service

import (
	"database/sql"
	"errors"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var ErrRolPermisoRolNoEncontrado = errors.New("rol no encontrado")

type RolPermisoService interface {
	ObtenerPermisosPorRol(codigoRol int64) (*dto.RolPermisosResponse, error)
}

type rolPermisoService struct {
	repository repository.RolPermisoRepository
}

func NewRolPermisoService(
	repository repository.RolPermisoRepository,
) RolPermisoService {
	return &rolPermisoService{
		repository: repository,
	}
}

func (s *rolPermisoService) ObtenerPermisosPorRol(
	codigoRol int64,
) (*dto.RolPermisosResponse, error) {
	response, err := s.repository.ObtenerPermisosPorRol(codigoRol)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRolPermisoRolNoEncontrado
		}

		return nil, err
	}

	return response, nil
}
