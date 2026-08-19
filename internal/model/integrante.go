package model

import "time"

type Integrante struct {
	CodIntegrante           int64      `json:"codIntegrante"`
	CodigoUsuario           int64      `json:"codigoUsuario"`
	NombreIntegrante        string     `json:"nombreIntegrante"`
	DescripcionIntegrante   *string    `json:"descripcionIntegrante"`
	FechaHoraBajaIntegrante *time.Time `json:"fechaHoraBajaIntegrante"`
}
