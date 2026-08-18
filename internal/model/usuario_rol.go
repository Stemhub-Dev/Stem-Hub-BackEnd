package model

import "time"

type UsuarioRol struct {
	CodigoUsuario           int64      `json:"codigoUsuario"`
	CodRol                  int64      `json:"codRol"`
	AmbitoRol               string     `json:"ambitoRol"`
	FechaHoraAltaUsuarioRol *time.Time `json:"fechaHoraAltaUsuarioRol"`
	FechaHoraBajaUsuarioRol *time.Time `json:"fechaHoraBajaUsuarioRol"`
}
