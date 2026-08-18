package model

import "time"

type Permiso struct {
	NombrePermiso        string     `json:"nombrePermiso"`
	DescripcionPermiso   string     `json:"descripcionPermiso"`
	AmbitoPermiso        string     `json:"ambitoPermiso"`
	FechaHoraBajaPermiso *time.Time `json:"FechaHoraBajaPermiso"`
	ClavePermiso         string     `json:"clavePermiso"`
}
