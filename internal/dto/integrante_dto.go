package dto

type ObtenerPerfilResponse struct {
	CodigoIntegrante       int64   `json:"codigoIntegrante"`
	Email                  string  `json:"email"`
	Nombre                 string  `json:"nombre"`
	Descripcion            *string `json:"descripcion"`
	AvatarUrl              *string `json:"avatarUrl"`
	EsAdministradorSistema bool    `json:"esAdministradorSistema"`
}

type EditarPerfilResponse struct {
	CodigoIntegrante int64   `json:"codigoIntegrante"`
	Email            string  `json:"email"`
	Nombre           string  `json:"nombre"`
	Descripcion      *string `json:"descripcion"`
	AvatarUrl        *string `json:"avatarUrl"`
}
