package model

import "time"

type IntegranteProyecto struct {
	CodigoIntegranteProyecto    int64      `json:"codigoIntegranteProyecto"`
	CodIntegrante               int64      `json:"codIntegrante"`
	CodigoProyecto              int64      `json:"codigoProyecto"`
	CodRol                      int64      `json:"codRol"`
	AmbitoRol                   string     `json:"ambitoRol"`
	EsPropietario               bool       `json:"esPropietario"`
	FechaHoraAltaIntegranteProy time.Time  `json:"fechaHoraAltaIntegranteProy"`
	FechaHoraBajaIntegranteProy *time.Time `json:"fechaHoraBajaIntegranteProy"`
}

// ColaboradorProyecto representa a un integrante activo de un proyecto tal
// como lo necesita el listado de colaboradores: sin resolver todavía la URL
// presignada del avatar (eso lo hace el service, que es quien tiene acceso
// al storage).
type ColaboradorProyecto struct {
	CodIntegrante   int64
	NombreIntegrante string
	AvatarObjectKey *string
	CodRol          int64
	NombreRol       string
	EsPropietario   bool
}
