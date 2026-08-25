package service

import (
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

type TipoProyectoService struct {
	repository *repository.TipoProyectoRepository
}

func NewTipoProyectoService(repository *repository.TipoProyectoRepository) *TipoProyectoService {
	return &TipoProyectoService{repository: repository}
}

func (s *TipoProyectoService) Listar() ([]dto.TipoProyectoResponse, error) {
	tipos, err := s.repository.Listar()
	if err != nil {
		return nil, err
	}

	respuesta := make([]dto.TipoProyectoResponse, 0, len(tipos))

	for _, tipo := range tipos {
		respuesta = append(respuesta, dto.TipoProyectoResponse{
			ID:     tipo.CodTipoProy,
			Nombre: tipo.NombreTipoProy,
		})
	}

	return respuesta, nil
}
