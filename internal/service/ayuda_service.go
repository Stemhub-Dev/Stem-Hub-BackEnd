package service

import (
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

type AyudaService interface {
	ListarPreguntasFrecuentes() ([]dto.PreguntaFrecuenteResponse, error)
}

type ayudaService struct {
	preguntas repository.PreguntaFrecuenteRepository
}

func NewAyudaService(preguntas repository.PreguntaFrecuenteRepository) AyudaService {
	return &ayudaService{preguntas: preguntas}
}

func (s *ayudaService) ListarPreguntasFrecuentes() ([]dto.PreguntaFrecuenteResponse, error) {
	preguntas, err := s.preguntas.ListarActivas()
	if err != nil {
		return nil, err
	}

	respuesta := make([]dto.PreguntaFrecuenteResponse, 0, len(preguntas))

	for _, pregunta := range preguntas {
		respuesta = append(respuesta, dto.PreguntaFrecuenteResponse{
			ID:                         pregunta.CodigoPreguntaFrecuente,
			FuncionalidadPrincipal:     pregunta.FuncionalidadPrincipal,
			PreguntaFrecuente:          pregunta.PreguntaFrecuente,
			RespuestaPreguntaFrecuente: pregunta.RespuestaFrecuente,
		})
	}

	return respuesta, nil
}
