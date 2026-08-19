package model

import "time"

type EstadoProyecto struct {
	CodEstadoProy           int64      `json:"codEstadoProy"`
	NombreEstadoProy        string     `json:"nombreEstadoProy"`
	DescripcionEstadoProy   *string    `json:"descripcionEstadoProy"`
	FechaHoraBajaEstadoProy *time.Time `json:"fechaHoraBajaEstadoProy"`
}
