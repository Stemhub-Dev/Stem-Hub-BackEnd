package dto

type ObtenerPerfilResponse struct {
	CodigoIntegrante int64   `json:"codigoIntegrante"`
	Email            string  `json:"email"`
	Nombre           string  `json:"nombre"`
	Descripcion      *string `json:"descripcion"`
}
