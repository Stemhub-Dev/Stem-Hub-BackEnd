package model

import "time"

type Cancion struct {
	CodigoCancion        int64      `json:"codigoCancion"`
	CodigoProyecto       int64      `json:"codigoProyecto"`
	NombreCancion        string     `json:"nombreCancion"`
	FechaHoraBajaCancion *time.Time `json:"fechaHoraBajaCancion"`
}
