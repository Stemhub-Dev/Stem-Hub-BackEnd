package dto

import "time"

type CrearInvitacionRequest struct {
	Email  string `json:"email" binding:"required,email"`
	CodRol int64  `json:"codRol" binding:"required"`
}

type InvitacionResponse struct {
	CodigoInvitacionProy int64     `json:"codigoInvitacionProy"`
	Email                string    `json:"email"`
	CodRol               int64     `json:"codRol"`
	NombreRol            string    `json:"nombreRol"`
	FechaHoraExpiracion  time.Time `json:"fechaHoraExpiracion"`
	Estado               string    `json:"estado"` // "pendiente" | "vencida"
}

// InvitacionDetalleResponse es la respuesta de GET /invitaciones/:token —
// lo que el frontend necesita ANTES de decidir aceptar, sin requerir
// sesión iniciada.
type InvitacionDetalleResponse struct {
	Email          string `json:"email"`
	NombreProyecto string `json:"nombreProyecto"`
	InvitadoPor    string `json:"invitadoPor"`
	NombreRol      string `json:"nombreRol"`
	Vencida        bool   `json:"vencida"`
	YaAceptada     bool   `json:"yaAceptada"`
	Cancelada      bool   `json:"cancelada"`
}

type AceptarInvitacionResponse struct {
	CodigoProyecto int64 `json:"codigoProyecto"`
}
