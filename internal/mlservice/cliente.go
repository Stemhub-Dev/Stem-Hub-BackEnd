// Package mlservice es el cliente HTTP de stemhub-microservicio-IA
// (separación de stems con Spleeter y resumen de comentarios con un LLM).
// Los tags JSON siguen el contrato del servicio Python (snake_case), no el
// camelCase de la API del backend.
package mlservice

import (
	"context"
	"fmt"
	"time"
)

type Cliente interface {
	// Bloquea hasta que el microservicio subió todos los stems a MinIO
	// (puede tardar minutos): llamarlo fuera de la request HTTP.
	SepararStems(ctx context.Context, request SeparacionRequest) (*SeparacionResponse, error)

	ResumirComentarios(ctx context.Context, request ResumenRequest) (*ResumenResponse, error)
}

// Nombres de stem que devuelve Spleeter según la cantidad pedida.
var NombresStemPorCantidad = map[int][]string{
	2: {"vocals", "accompaniment"},
	4: {"vocals", "drums", "bass", "other"},
	5: {"vocals", "drums", "bass", "piano", "other"},
}

// SeparacionRequest: el microservicio lee SourceObjectKey del bucket
// compartido y sube cada stem a la key de DestinationObjectKeys, que debe
// tener exactamente los nombres de NombresStemPorCantidad[StemCount].
type SeparacionRequest struct {
	SourceObjectKey       string            `json:"source_object_key"`
	StemCount             int               `json:"stem_count"`
	DestinationObjectKeys map[string]string `json:"destination_object_keys"`
}

type StemSeparado struct {
	StemName        string  `json:"stem_name"`
	ObjectKey       string  `json:"object_key"`
	DurationSeconds float64 `json:"duration_seconds"`
}

type SeparacionResponse struct {
	Status           string         `json:"status"`
	Stems            []StemSeparado `json:"stems"`
	ProcessingTimeMs int64          `json:"processing_time_ms"`
}

type ComentarioResumen struct {
	Author    string     `json:"author"`
	Text      string     `json:"text"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

type ContextoResumen struct {
	SongName     string `json:"song_name,omitempty"`
	VersionLabel string `json:"version_label,omitempty"`
}

type ResumenRequest struct {
	Context  *ContextoResumen    `json:"context,omitempty"`
	Comments []ComentarioResumen `json:"comments"`
}

type ResumenResponse struct {
	ProviderUsed     string   `json:"provider_used"`
	Summary          string   `json:"summary"`
	KeyPoints        []string `json:"key_points"`
	Agreements       []string `json:"agreements"`
	Disagreements    []string `json:"disagreements"`
	OverallSentiment string   `json:"overall_sentiment"`
	ProcessingTimeMs int64    `json:"processing_time_ms"`
}

// ErrorServicioML es una respuesta no 2xx del microservicio. Codigo es su
// campo "error" (ej. LLM_TIMEOUT, SEPARATION_WORKER_ERROR); vacío si el
// cuerpo no tenía ese formato (ej. un 422 de validación de FastAPI).
type ErrorServicioML struct {
	Status  int
	Codigo  string
	Mensaje string
	// Campo "detail" crudo (ej. el motivo que dio Gemini): solo para el log.
	Detalle string
}

func (e *ErrorServicioML) Error() string {
	mensaje := fmt.Sprintf("servicio de IA respondió %d %s: %s", e.Status, e.Codigo, e.Mensaje)

	if e.Detalle != "" {
		mensaje += " (detalle: " + e.Detalle + ")"
	}

	return mensaje
}

// Códigos de error del microservicio que el backend distingue.
const (
	CodigoLLMTimeout        = "LLM_TIMEOUT"
	CodigoSeparacionTimeout = "SEPARATION_TIMEOUT"
)
