package dto

type CrearPerfilIntegranteRequest struct {
	Nombre      string  `json:"nombre" binding:"required"`
	Descripcion *string `json:"descripcion"`
}

type ObtenerPerfilResponse struct {
	CodigoIntegrante int64   `json:"codigoIntegrante"`
	Email            string  `json:"email"`
	Nombre           string  `json:"nombre"`
	Descripcion      *string `json:"descripcion"`
}
