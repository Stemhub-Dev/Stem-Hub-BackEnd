package dto

type PermisoRolResponse struct {
	CodigoPermiso      int64   `json:"codigoPermiso"`
	ClavePermiso       string  `json:"clavePermiso"`
	NombrePermiso      string  `json:"nombrePermiso"`
	DescripcionPermiso *string `json:"descripcionPermiso"`
	Asignado           bool    `json:"asignado"`
}

type RolPermisosResponse struct {
	CodigoRol int64                `json:"codigoRol"`
	NombreRol string               `json:"nombreRol"`
	AmbitoRol string               `json:"ambitoRol"`
	Permisos  []PermisoRolResponse `json:"permisos"`
}
