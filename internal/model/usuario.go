package model

import "time"

type Usuario struct {
	CodigoUsuario        int64      `json:"codigoUsuario"`
	IDAutenticacion      string     `json:"-"`
	Email                string     `json:"email"`
	UltimoLogin          *time.Time `json:"ultimoLogin"`
	FechaHoraBajaUsuario *time.Time `json:"fechaHoraBajaUsuario"`
}
