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

type ColaboradorProyectoResponse struct {
	CodigoIntegrante int64   `json:"codigoIntegrante"`
	Nombre           string  `json:"nombre"`
	AvatarUrl        *string `json:"avatarUrl"`
	CodRol           int64   `json:"codRol"`
	NombreRol        string  `json:"nombreRol"`
	EsPropietario    bool    `json:"esPropietario"`
}

type GeneroProyectoResponse struct {
	CodigoGenero int64  `json:"codigoGenero"`
	NombreGenero string `json:"nombreGenero"`
}

type ProyectoDetalleResponse struct {
	CodigoProyecto int64   `json:"codigoProyecto"`
	Nombre         string  `json:"nombre"`
	Descripcion    *string `json:"descripcion"`

	LogoUrl *string `json:"logoUrl"`

	CodigoTipoProyecto int64  `json:"codigoTipoProyecto"`
	NombreTipoProyecto string `json:"nombreTipoProyecto"`

	CodigoEstadoProyecto int64  `json:"codigoEstadoProyecto"`
	NombreEstadoProyecto string `json:"nombreEstadoProyecto"`

	Generos []GeneroProyectoResponse `json:"generos"`

	EsPropietario bool `json:"esPropietario"`
}

type EditarProyectoRequest struct {
	Nombre               *string `json:"nombre" form:"nombre"`
	Descripcion          *string `json:"descripcion" form:"descripcion"`
	CodigoTipoProyecto   *int64  `json:"codigoTipoProyecto" form:"codigoTipoProyecto"`
	CodigoEstadoProyecto *int64  `json:"codigoEstadoProyecto" form:"codigoEstadoProyecto"`
	CodigosGeneros       []int64 `json:"codigosGeneros" form:"codigosGeneros"`
}

type EditarProyectoResponse struct {
	CodigoProyecto int64   `json:"codigoProyecto"`
	Nombre         string  `json:"nombre"`
	Descripcion    *string `json:"descripcion"`

	LogoUrl *string `json:"logoUrl"`

	CodigoTipoProyecto   int64   `json:"codigoTipoProyecto"`
	CodigoEstadoProyecto int64   `json:"codigoEstadoProyecto"`
	CodigosGeneros       []int64 `json:"codigosGeneros"`
}
