package service

import "github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"

type PermisoService interface {
	TienePermiso(codigoUsuario int64, clavePermiso string) (bool, error)
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
