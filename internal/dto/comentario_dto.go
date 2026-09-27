package dto

import "time"

type CrearComentarioRequest struct {
	Texto                string   `json:"texto" binding:"required,max=200"`
	TiempoInicioSegundos *float64 `json:"tiempoInicioSegundos" binding:"omitempty,min=0"`
	TiempoFinSegundos    *float64 `json:"tiempoFinSegundos" binding:"omitempty,min=0"`
	// Opcional: el comentario es de ese stem de la versión. Sin él, es de
	// la versión completa.
	CodStem *int64 `json:"codStem" binding:"omitempty,min=1"`
}

type AutorComentarioResponse struct {
	CodigoIntegrante int64  `json:"codigoIntegrante"`
	Nombre           string `json:"nombre"`
}

type CrearComentarioResponse struct {
	CodigoComentario     int64                   `json:"codigoComentario"`
	CodStem              *int64                  `json:"codStem"`
	Texto                string                  `json:"texto"`
	Estado               string                  `json:"estado"`
	TiempoInicioSegundos *float64                `json:"tiempoInicioSegundos"`
	TiempoFinSegundos    *float64                `json:"tiempoFinSegundos"`
	Autor                AutorComentarioResponse `json:"autor"`
}

type ComentarioListadoResponse struct {
	CodigoComentario     int64                         `json:"codigoComentario"`
	CodStem              *int64                        `json:"codStem"`
	Texto                string                        `json:"texto"`
	Estado               string                        `json:"estado"`
	TiempoInicioSegundos *float64                      `json:"tiempoInicioSegundos"`
	TiempoFinSegundos    *float64                      `json:"tiempoFinSegundos"`
	FechaHoraAlta        time.Time                     `json:"fechaHoraAlta"`
	Autor                AutorComentarioResponse       `json:"autor"`
	EsPropio             bool                          `json:"esPropio"`
	Respuestas           []RespuestaComentarioResponse `json:"respuestas"`
}

type CrearRespuestaComentarioRequest struct {
	Texto string `json:"texto" binding:"required,max=200"`
}

type RespuestaComentarioResponse struct {
	CodigoRespuesta int64                   `json:"codigoRespuesta"`
	Texto           string                  `json:"texto"`
	FechaHoraAlta   time.Time               `json:"fechaHoraAlta"`
	Autor           AutorComentarioResponse `json:"autor"`
	EsPropia        bool                    `json:"esPropia"`
}

type ModificarComentarioRequest struct {
	Texto string `json:"texto" binding:"required,max=200"`
}

type ModificarComentarioResponse struct {
	CodigoComentario     int64                   `json:"codigoComentario"`
	Texto                string                  `json:"texto"`
	Estado               string                  `json:"estado"`
	TiempoInicioSegundos *float64                `json:"tiempoInicioSegundos"`
	TiempoFinSegundos    *float64                `json:"tiempoFinSegundos"`
	Autor                AutorComentarioResponse `json:"autor"`
}

type CambiarEstadoComentarioRequest struct {
	Estado string `json:"estado" binding:"required"`
}

type CambiarEstadoComentarioResponse struct {
	CodigoComentario int64  `json:"codigoComentario"`
	Estado           string `json:"estado"`
}
