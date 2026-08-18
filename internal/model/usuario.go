package model

import "time"

type Usuario struct {
	CodigoUsuario        int64      `json:"codigoUsuario"`
	Email                string     `json:"email"`
	ContrasenaHash       string     `json:"-"`
	UltimoLogin          *time.Time `json:"ultimoLogin"`
	FechaHoraBajaUsuario *time.Time `json:"fechaHoraBajaUsuario"`
}
