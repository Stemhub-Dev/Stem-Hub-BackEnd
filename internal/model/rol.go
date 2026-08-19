package model

import "time"

type Rol struct {
	//el asterisco indica que el campo puede ser nulo
	//el json es como se debe escribir en el frontend
	CodRol           int64      `json:"codRol"`
	NombreRol        string     `json:"nombreRol"`
	DescripcionRol   *string    `json:"descripcionRol"`
	AmbitoRol        string     `json:"ambitoRol"`
	FechaHoraBajaRol *time.Time `json:"fechaHoraBajaRol"`
}
