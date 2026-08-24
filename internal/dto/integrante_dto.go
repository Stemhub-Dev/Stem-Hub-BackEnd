package dto

type CrearPerfilIntegranteRequest struct {
	Nombre      string  `json:"nombre" binding:"required"`
	Descripcion *string `json:"descripcion"`
}
