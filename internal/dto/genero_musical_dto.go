package dto

type GeneroMusicalResponse struct {
	ID     int64  `json:"id"`
	Nombre string `json:"nombre"`
	Activo bool   `json:"activo"`
}

type GeneroMusicalRequest struct {
	Nombre string `json:"nombre"`
}

type CambiarEstadoGeneroMusicalRequest struct {
	Activo *bool `json:"activo"`
}
