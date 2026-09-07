package dto

import "time"

type CrearComentarioRequest struct {
	Texto string `json:"texto" binding:"required,max=200"`
}

type AutorComentarioResponse struct {
	CodigoIntegrante int64  `json:"codigoIntegrante"`
	Nombre           string `json:"nombre"`
}

type CrearComentarioResponse struct {
	CodigoComentario int64                   `json:"codigoComentario"`
	Texto            string                  `json:"texto"`
	Estado           string                  `json:"estado"`
	Autor            AutorComentarioResponse `json:"autor"`
}

type ComentarioListadoResponse struct {
	CodigoComentario int64                   `json:"codigoComentario"`
	Texto            string                  `json:"texto"`
	Estado           string                  `json:"estado"`
	FechaHoraAlta    time.Time               `json:"fechaHoraAlta"`
	Autor            AutorComentarioResponse `json:"autor"`
	EsPropio         bool                    `json:"esPropio"`
}
