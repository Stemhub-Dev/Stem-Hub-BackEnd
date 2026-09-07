package service

import (
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

type PermisoService interface {
	TienePermiso(codigoUsuario int64, clavePermiso string) (bool, error)
	ListarActivos() ([]model.Permiso, error)
}

type permisoService struct {
	repository repository.PermisoRepository
}

func NewPermisoService(repository repository.PermisoRepository) PermisoService {
	return &permisoService{
		repository: repository,
	}
}

func (s *permisoService) TienePermiso(
	codigoUsuario int64,
	clavePermiso string,
) (bool, error) {
	return s.repository.TienePermiso(
		codigoUsuario,
		clavePermiso,
	)
}

func (s *permisoService) ListarActivos() (
	[]model.Permiso,
	error,
) {
	return s.repository.ListarActivos()
}
