package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/mlservice"
)

var (
	ErrResumenSinComentarios = errors.New("la versión no tiene comentarios para resumir")
	ErrResumenTimeout        = errors.New("el servicio de IA tardó demasiado en generar el resumen")
	ErrResumenNoDisponible   = errors.New("el servicio de IA no pudo generar el resumen")
)

func (s *comentarioService) ResumirPorVersion(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
) (*dto.ResumenComentariosResponse, error) {

	integrante, err := s.validarAccesoVersion(
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
	)

	if err != nil {
		return nil, err
	}

	comentarios, err := s.comentarioRepository.ListarPorVersion(
		codigoVersion,
		integrante.CodIntegrante,
	)

	if err != nil {
		return nil, err
	}

	if len(comentarios) == 0 {
		return nil, ErrResumenSinComentarios
	}

	nombreCancion, numeroVersion, err := s.comentarioRepository.ObtenerContextoVersion(codigoVersion)

	if err != nil {
		return nil, err
	}

	entrada := comentariosParaResumen(comentarios)

	resultado, err := s.mlCliente.ResumirComentarios(context.Background(), mlservice.ResumenRequest{
		Context: &mlservice.ContextoResumen{
			SongName:     nombreCancion,
			VersionLabel: fmt.Sprintf("v%d", numeroVersion),
		},
		Comments: entrada,
	})

	if err != nil {
		log.Printf("Falló el resumen de comentarios de la versión %d: %v", codigoVersion, err)

		var errorML *mlservice.ErrorServicioML

		if errors.Is(err, context.DeadlineExceeded) ||
			(errors.As(err, &errorML) && errorML.Codigo == mlservice.CodigoLLMTimeout) {
			return nil, ErrResumenTimeout
		}

		return nil, ErrResumenNoDisponible
	}

	return &dto.ResumenComentariosResponse{
		Resumen:               resultado.Summary,
		PuntosClave:           noNulo(resultado.KeyPoints),
		Acuerdos:              noNulo(resultado.Agreements),
		Desacuerdos:           noNulo(resultado.Disagreements),
		SentimientoGeneral:    resultado.OverallSentiment,
		CantidadComentarios:   len(entrada),
		Proveedor:             resultado.ProviderUsed,
		TiempoProcesamientoMs: resultado.ProcessingTimeMs,
	}, nil
}

// comentariosParaResumen aplana comentarios y respuestas en orden
// cronológico (el listado viene del más nuevo al más viejo). El rango de
// tiempo y a quién se responde van en el texto para que el LLM los tenga en
// cuenta.
func comentariosParaResumen(comentarios []dto.ComentarioListadoResponse) []mlservice.ComentarioResumen {

	entrada := []mlservice.ComentarioResumen{}

	for i := len(comentarios) - 1; i >= 0; i-- {

		comentario := comentarios[i]
		texto := comentario.Texto

		if comentario.TiempoInicioSegundos != nil {
			rango := formatearSegundos(*comentario.TiempoInicioSegundos)

			if comentario.TiempoFinSegundos != nil && *comentario.TiempoFinSegundos != *comentario.TiempoInicioSegundos {
				rango += "–" + formatearSegundos(*comentario.TiempoFinSegundos)
			}

			texto = "[" + rango + "] " + texto
		}

		entrada = append(entrada, mlservice.ComentarioResumen{
			Author:    comentario.Autor.Nombre,
			Text:      texto,
			Timestamp: horaUTC(comentario.FechaHoraAlta),
		})

		for _, respuesta := range comentario.Respuestas {
			entrada = append(entrada, mlservice.ComentarioResumen{
				Author:    respuesta.Autor.Nombre,
				Text:      fmt.Sprintf("(respuesta a %s) %s", comentario.Autor.Nombre, respuesta.Texto),
				Timestamp: horaUTC(respuesta.FechaHoraAlta),
			})
		}
	}

	return entrada
}

func formatearSegundos(segundos float64) string {
	total := int(segundos)
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

func horaUTC(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	utc := t.UTC()
	return &utc
}

// Para que el JSON devuelva [] y no null si el LLM omite una lista.
func noNulo(valores []string) []string {
	if valores == nil {
		return []string{}
	}
	return valores
}
