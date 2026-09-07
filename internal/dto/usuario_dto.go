package dto

type RegistrarUsuarioRequest struct {
	Nombre      string  `json:"nombre" binding:"required"`
	Descripcion *string `json:"descripcion"`
}

type RegistrarUsuarioResponse struct {
	CodigoUsuario    int64   `json:"codigoUsuario"`
	Email            string  `json:"email"`
	CodigoIntegrante int64   `json:"codigoIntegrante"`
	Nombre           string  `json:"nombre"`
	Descripcion      *string `json:"descripcion"`
}
