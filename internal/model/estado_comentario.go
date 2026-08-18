package model

import "time"

type EstadoComentario struct {
	CodEstadoCom           int64      `json:"codEstadoCom"`
	NombreEstadoCom        string     `json:"nombreEstadoCom"`
	DescripcionEstadoCom   *string    `json:"descripcionEstadoCom"`
	FechaHoraBajaEstadoCom *time.Time `json:"fechaHoraBajaEstadoCom"`
}
