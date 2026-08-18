package model

import "time"

type PreguntaFrecuente struct {
	CodigoPreguntaFrecuente        int64      `json:"codigoPreguntaFrecuente"`
	FuncionalidadPrincipal         string     `json:"funcionalidadPrincipal"`
	PreguntaFrecuente              string     `json:"preguntaFrecuente"`
	RespuestaFrecuente             string     `json:"respuestaFrecuente"`
	FechaHoraBajaPreguntaFrecuente *time.Time `json:"fechaHoraBajaPreguntaFrecuente"`
}
