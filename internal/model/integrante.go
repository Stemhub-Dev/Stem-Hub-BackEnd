package model

import "time"

type Integrante struct {
	CodIntegrante           int64      `json:"codIntegrante"`
	CodigoUsuario           int64      `json:"-"`
	NombreIntegrante        string     `json:"nombreIntegrante"`
	DescripcionIntegrante   *string    `json:"descripcionIntegrante"`
	AvatarObjectKey         *string    `json:"-"`
	FechaHoraBajaIntegrante *time.Time `json:"-"`
}
