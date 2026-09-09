package dto

import "time"

type RolSistemaUsuarioResponse struct {
	CodRol    int64  `json:"codRol"`
	NombreRol string `json:"nombreRol"`
}

type UsuarioAdministracionResponse struct {
	CodigoUsuario         int64                       `json:"codigoUsuario"`
	CodIntegrante         int64                       `json:"codIntegrante"`
	Email                 string                      `json:"email"`
	NombreIntegrante      string                      `json:"nombreIntegrante"`
	DescripcionIntegrante *string                     `json:"descripcionIntegrante"`
	UltimoLogin           *time.Time                  `json:"ultimoLogin"`
	Activo                bool                        `json:"activo"`
	RolesSistema          []RolSistemaUsuarioResponse `json:"rolesSistema"`
}

type RolUsuarioAdministracionResponse struct {
	CodRol         int64   `json:"codRol"`
	NombreRol      string  `json:"nombreRol"`
	DescripcionRol *string `json:"descripcionRol"`
	Asignado       bool    `json:"asignado"`
}

type UsuarioRolesResponse struct {
	CodigoUsuario int64                              `json:"codigoUsuario"`
	Roles         []RolUsuarioAdministracionResponse `json:"roles"`
}

type ProyectoUsuarioResponse struct {
	CodigoProyecto int64  `json:"codigoProyecto"`
	NombreProyecto string `json:"nombreProyecto"`
	CodRol         int64  `json:"codRol"`
	NombreRol      string `json:"nombreRol"`
	EsPropietario  bool   `json:"esPropietario"`
}

type UsuarioProyectosResponse struct {
	CodigoUsuario int64                     `json:"codigoUsuario"`
	Proyectos     []ProyectoUsuarioResponse `json:"proyectos"`
}
