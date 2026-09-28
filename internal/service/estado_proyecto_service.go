package service

import (
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

type EstadoProyectoService struct {
	repository *repository.EstadoProyectoRepository
}

func NewEstadoProyectoService(
	repository *repository.EstadoProyectoRepository,
) *EstadoProyectoService {

	return &EstadoProyectoService{
		repository: repository,
	}
}

func (s *EstadoProyectoService) Listar() (
	[]dto.EstadoProyectoResponse,
	error,
) {

	estados, err := s.repository.Listar()

	if err != nil {
		return nil, err
	}

	respuesta := make(
		[]dto.EstadoProyectoResponse,
		0,
		len(estados),
	)

	for _, estado := range estados {

		respuesta = append(
			respuesta,
			dto.EstadoProyectoResponse{
				ID:     estado.CodEstadoProy,
				Nombre: estado.NombreEstadoProy,
			},
		)
	}

	return respuesta, nil
}
