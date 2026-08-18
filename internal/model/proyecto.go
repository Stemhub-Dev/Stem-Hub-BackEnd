package model

import "time"

type Proyecto struct {
	CodigoProyecto        int64      `json:"codigoProyecto"`
	NombreProyecto        string     `json:"nombreProyecto"`
	DescripcionProyecto   *string    `json:"descripcionProyecto"`
	LogoProyecto          *string    `json:"logoProyecto"`
	CodEstadoProy         int64      `json:"codEstadoProy"`
	CodTipoProy           int64      `json:"codTipoProy"`
	FechaHoraBajaProyecto *time.Time `json:"fechaHoraBajaProyecto"`
}
