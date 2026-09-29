package dto

type TipoProyectoResponse struct {
	ID     int64  `json:"id"`
	Nombre string `json:"nombre"`
	Activo bool   `json:"activo"`
}

type TipoProyectoRequest struct {
	Nombre string `json:"nombre"`
}

type CambiarEstadoTipoProyectoRequest struct {
	Activo *bool `json:"activo"`
}
