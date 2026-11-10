package service

import (
	"errors"
	"io/fs"
	"os"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// ErrManualUsuarioNoDisponible indica que el manual no está cargado.
var ErrManualUsuarioNoDisponible = errors.New("No está cargado el manual de usuario")

type AyudaService interface {
	ListarPreguntasFrecuentes() ([]dto.PreguntaFrecuenteResponse, error)
	ObtenerRutaManual() (string, error)
}

type ayudaService struct {
	preguntas  repository.PreguntaFrecuenteRepository
	rutaManual string
}

func NewAyudaService(
	preguntas repository.PreguntaFrecuenteRepository,
	rutaManual string,
) AyudaService {
	return &ayudaService{preguntas: preguntas, rutaManual: rutaManual}
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

// ObtenerRutaManual se fija en cada pedido si el archivo existe, para que
// reemplazarlo o quitarlo no requiera reiniciar el servicio.
func (s *ayudaService) ObtenerRutaManual() (string, error) {
	info, err := os.Stat(s.rutaManual)

	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrManualUsuarioNoDisponible
	}
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", ErrManualUsuarioNoDisponible
	}

	return s.rutaManual, nil
}
