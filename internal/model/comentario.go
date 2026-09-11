package model

import "time"

type Comentario struct {
	CodigoComentario        int64      `json:"codigoComentario"`
	CodIntegrante           int64      `json:"codIntegrante"`
	CodEstadoCom            int64      `json:"codEstadoCom"`
	CodigoProyecto          *int64     `json:"codigoProyecto"`
	CodigoCancionVersion    *int64     `json:"codigoCancionVersion"`
	DescripcionComentario   string     `json:"descripcionComentario"`
	TiempoInicioSegundos    *float64   `json:"tiempoInicioSegundos"`
	TiempoFinSegundos       *float64   `json:"tiempoFinSegundos"`
	FechaHoraAltaComentario time.Time  `json:"fechaHoraAltaComentario"`
	FechaHoraBajaComentario *time.Time `json:"fechaHoraBajaComentario"`
}
