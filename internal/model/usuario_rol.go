package model

import "time"

type UsuarioRol struct {
	CodigoUsuarioRol        int64      `json:"codigoUsuarioRol"`
	CodigoUsuario           int64      `json:"codigoUsuario"`
	CodRol                  int64      `json:"codRol"`
	AmbitoRol               string     `json:"ambitoRol"`
	FechaHoraAltaUsuarioRol time.Time  `json:"fechaHoraAltaUsuarioRol"`
	FechaHoraBajaUsuarioRol *time.Time `json:"fechaHoraBajaUsuarioRol"`
}
