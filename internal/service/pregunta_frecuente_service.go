package service

import (
	"database/sql"
	"errors"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrPreguntaFrecuenteNoEncontrada = errors.New("pregunta frecuente no encontrada")
)

type PreguntaFrecuenteService struct {
	repository *repository.PreguntaFrecuenteRepository
}

func NewPreguntaFrecuenteService(repository *repository.PreguntaFrecuenteRepository) *PreguntaFrecuenteService {
	return &PreguntaFrecuenteService{
		repository: repository,
	}
}

func (s *PreguntaFrecuenteService) Listar() (
	[]dto.PreguntaFrecuenteResponse, error) {
	preguntas, err := s.repository.Listar()
	if err != nil {
		return nil, err
	}

	response := make([]dto.PreguntaFrecuenteResponse, 0, len(preguntas))

	for _, pregunta := range preguntas {
		response = append(response, dto.PreguntaFrecuenteResponse{
			ID:                         pregunta.CodigoPreguntaFrecuente,
			FuncionalidadPrincipal:     pregunta.FuncionalidadPrincipal,
			PreguntaFrecuente:          pregunta.PreguntaFrecuente,
			RespuestaPreguntaFrecuente: pregunta.RespuestaFrecuente,
		})
	}

	return response, nil
}

func (s *PreguntaFrecuenteService) ObtenerPorID(id int64) (*dto.PreguntaFrecuenteResponse, error) {
	pregunta, err := s.repository.BuscarPorID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPreguntaFrecuenteNoEncontrada
		}
		return nil, err
	}

	return &dto.PreguntaFrecuenteResponse{
		ID:                         pregunta.CodigoPreguntaFrecuente,
		FuncionalidadPrincipal:     pregunta.FuncionalidadPrincipal,
		PreguntaFrecuente:          pregunta.PreguntaFrecuente,
		RespuestaPreguntaFrecuente: pregunta.RespuestaFrecuente,
	}, nil
}

