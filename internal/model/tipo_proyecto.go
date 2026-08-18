package model

import "time"

type TipoProyecto struct {
	CodTipoProy           int64      `json:"codTipoProy"`
	NombreTipoProy        string     `json:"nombreTipoProy"`
	DescripcionTipoProy   *string    `json:"descripcionTipoProy"`
	FechaHoraBajaTipoProy *time.Time `json:"fechaHoraBajaTipoProy"`
}
