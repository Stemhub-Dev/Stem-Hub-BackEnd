package dto

type EstadoProyectoResponse struct {
	ID     int64  `json:"id"`
	Nombre string `json:"nombre"`
	Activo bool   `json:"activo"`
}

type EstadoProyectoRequest struct {
	Nombre string `json:"nombre"`
}

type CambiarEstadoProyectoRequest struct {
	Activo *bool `json:"activo"`
}
