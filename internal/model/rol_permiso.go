package model

import "time"

type RolPermiso struct {
	CodRol                  int64      `json:"codRol"`
	CodigoPermiso           int64      `json:"codigoPermiso"`
	AmbitoRolPermiso        string     `json:"ambitoRolPermiso"`
	FechaHoraAltaRolPermiso *time.Time `json:"fechaHoraAltaRolPermiso"`
	FechaHoraBajaRolPermiso *time.Time `json:"fechaHoraBajaRolPermiso"`
}
