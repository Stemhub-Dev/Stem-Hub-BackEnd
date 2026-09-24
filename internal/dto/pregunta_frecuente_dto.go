package dto

type PreguntaFrecuenteResponse struct {
	ID                         int64  `json:"id"`
	FuncionalidadPrincipal     string `json:"funcionalidadPrincipal"`
	PreguntaFrecuente          string `json:"preguntaFrecuente"`
	RespuestaPreguntaFrecuente string `json:"respuestaPreguntaFrecuente"`
}

