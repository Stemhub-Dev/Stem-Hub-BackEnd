package model

import "time"

type Permiso struct {
	CodigoPermiso        int64      `json:"codigoPermiso"`
	NombrePermiso        string     `json:"nombrePermiso"`
	DescripcionPermiso   *string    `json:"descripcionPermiso"`
	AmbitoPermiso        string     `json:"ambitoPermiso"`
	FechaHoraBajaPermiso *time.Time `json:"fechaHoraBajaPermiso"`
	ClavePermiso         string     `json:"clavePermiso"`
}
