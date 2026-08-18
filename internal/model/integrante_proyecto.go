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
