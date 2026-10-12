package dto

type PermisoResponse struct {
	CodigoPermiso      int64   `json:"codigoPermiso"`
	NombrePermiso      string  `json:"nombrePermiso"`
	DescripcionPermiso *string `json:"descripcionPermiso"`
	AmbitoPermiso      string  `json:"ambitoPermiso"`
	ClavePermiso       string  `json:"clavePermiso"`
	Activo             bool    `json:"activo"`
}

type PermisoRequest struct {
	NombrePermiso      string  `json:"nombrePermiso"`
	DescripcionPermiso *string `json:"descripcionPermiso"`
	AmbitoPermiso      string  `json:"ambitoPermiso"`
	ClavePermiso       string  `json:"clavePermiso"`
}

type CambiarEstadoPermisoRequest struct {
	Activo *bool `json:"activo"`
}
