package model

import "time"

type GeneroMusicalProyecto struct {
	CodigoGeneroProy        int64      `json:"codigoGeneroProy"`
	NombreGeneroProy        string     `json:"nombreGeneroProy"`
	DescripcionGeneroProy   *string    `json:"descripcionGeneroProy"`
	FechaHoraBajaGeneroProy *time.Time `json:"fechaHoraBajaGeneroProy"`
}
