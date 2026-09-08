package dto

import "time"

type CrearComentarioRequest struct {
	Texto                string   `json:"texto" binding:"required,max=200"`
	TiempoInicioSegundos *float64 `json:"tiempoInicioSegundos" binding:"omitempty,min=0"`
	TiempoFinSegundos    *float64 `json:"tiempoFinSegundos" binding:"omitempty,min=0"`
}

type AutorComentarioResponse struct {
	CodigoIntegrante int64  `json:"codigoIntegrante"`
	Nombre           string `json:"nombre"`
}

type CrearComentarioResponse struct {
	CodigoComentario     int64                   `json:"codigoComentario"`
	Texto                string                  `json:"texto"`
	Estado               string                  `json:"estado"`
	TiempoInicioSegundos *float64                `json:"tiempoInicioSegundos"`
	TiempoFinSegundos    *float64                `json:"tiempoFinSegundos"`
	Autor                AutorComentarioResponse `json:"autor"`
}

type ComentarioListadoResponse struct {
	CodigoComentario     int64                   `json:"codigoComentario"`
	Texto                string                  `json:"texto"`
	Estado               string                  `json:"estado"`
	TiempoInicioSegundos *float64                `json:"tiempoInicioSegundos"`
	TiempoFinSegundos    *float64                `json:"tiempoFinSegundos"`
	FechaHoraAlta        time.Time               `json:"fechaHoraAlta"`
	Autor                AutorComentarioResponse `json:"autor"`
	EsPropio             bool                    `json:"esPropio"`
}
