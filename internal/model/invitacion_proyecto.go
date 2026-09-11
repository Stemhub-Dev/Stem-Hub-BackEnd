package model

import "time"

type InvitacionProyecto struct {
	CodigoInvitacionProy        int64      `json:"codigoInvitacionProy"`
	CodigoProyecto              int64      `json:"codigoProyecto"`
	EmailInvitado               string     `json:"emailInvitado"`
	CodRol                      int64      `json:"codRol"`
	AmbitoRol                   string     `json:"ambitoRol"`
	CodIntegranteInvito         int64      `json:"-"`
	TokenInvitacion             string     `json:"-"`
	FechaHoraAltaInvitacionProy time.Time  `json:"fechaHoraAltaInvitacionProy"`
	FechaHoraExpiracion         time.Time  `json:"fechaHoraExpiracion"`
	FechaHoraAceptacion         *time.Time `json:"fechaHoraAceptacion"`
	FechaHoraBajaInvitacionProy *time.Time `json:"fechaHoraBajaInvitacionProy"`
}

// InvitacionDetalle es lo que necesita la pantalla de aceptación del
// frontend antes de decidir aceptar: quién invita y a qué proyecto, sin
// requerir sesión iniciada.
type InvitacionDetalle struct {
	EmailInvitado          string
	NombreProyecto         string
	NombreIntegranteInvito string
	NombreRol              string
	Vencida                bool
	YaAceptada             bool
	Cancelada              bool
}
