package dto

type CrearProyectoRequest struct {
	Nombre             string  `json:"nombre" binding:"required"`
	Descripcion        *string `json:"descripcion"`
	CodigoTipoProyecto int64   `json:"codigoTipoProyecto" binding:"required"`
	CodigosGeneros     []int64 `json:"codigosGeneros" binding:"required,min=1"`
	CodRol             int64   `json:"codRol" binding:"required"`
}

type CrearProyectoResponse struct {
	CodigoProyecto     int64   `json:"codigoProyecto"`
	Nombre             string  `json:"nombre"`
	Descripcion        *string `json:"descripcion"`
	CodigoTipoProyecto int64   `json:"codigoTipoProyecto"`
	CodigosGeneros     []int64 `json:"codigosGeneros"`
	CodRol             int64   `json:"codRol"`
	EsPropietario      bool    `json:"esPropietario"`
}

type ProyectoListadoResponse struct {
	CodigoProyecto    int64   `json:"codigoProyecto"`
	Nombre            string  `json:"nombre"`
	Descripcion       *string `json:"descripcion"`
	Logo              *string `json:"logo"`
	Tipo              string  `json:"tipo"`
	Estado            string  `json:"estado"`
	CantidadCanciones int     `json:"cantidadCanciones"`

	CodRol        int64  `json:"codRol"`
	NombreRol     string `json:"nombreRol"`
	EsPropietario bool   `json:"esPropietario"`
}
