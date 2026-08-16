package service

import (
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

type RolService struct {
	repository *repository.RolRepository
}

func NewRolService(repository *repository.RolRepository) *RolService {
	return &RolService{
		repository: repository,
	}
}

func (s *RolService) Listar() ([]model.Rol, error) {
	return s.repository.Listar()
}
