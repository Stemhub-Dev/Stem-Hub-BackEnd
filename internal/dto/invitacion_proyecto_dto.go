package dto

import "time"

// El formato del email lo valida el service, para responder "El email es
// inválido" (HU-NOT-B01) en vez del genérico del binding.
type CrearInvitacionRequest struct {
	Email  string `json:"email"`
	CodRol int64  `json:"codRol" binding:"required"`
}

// CrearEnlaceInvitacionRequest es el body de
// POST /proyectos/:id/invitaciones/enlace: solo el rol, sin email.
type CrearEnlaceInvitacionRequest struct {
	CodRol int64 `json:"codRol" binding:"required"`
}

// InvitacionResponse la ve solo el propietario del proyecto. Enlace es el
// link de aceptación: lo copia para compartirlo a mano (invitación por
// enlace, o si el mail no salió).
type InvitacionResponse struct {
	CodigoInvitacionProy int64     `json:"codigoInvitacionProy"`
	Email                string    `json:"email"` // "" en las invitaciones por enlace
	PorEnlace            bool      `json:"porEnlace"`
	CodRol               int64     `json:"codRol"`
	NombreRol            string    `json:"nombreRol"`
	FechaHoraExpiracion  time.Time `json:"fechaHoraExpiracion"`
	Estado               string    `json:"estado"` // "pendiente" | "vencida"
	Enlace               string    `json:"enlace"`
	// Solo al crear por email: false si falló el envío del mail (la
	// invitación igual queda creada).
	EmailEnviado *bool `json:"emailEnviado,omitempty"`
}

// InvitacionDetalleResponse es la respuesta de GET /invitaciones/:token —
// lo que el frontend necesita ANTES de decidir aceptar, sin requerir
// sesión iniciada.
type InvitacionDetalleResponse struct {
	Email          string `json:"email"` // "" si es por enlace: la acepta cualquier cuenta
	PorEnlace      bool   `json:"porEnlace"`
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

// InvitacionPendienteResponse es una fila de la bandeja de notificaciones
// del usuario invitado (GET /invitaciones). Incluye el token para poder
// aceptar/rechazar inline sin un segundo viaje a buscarlo.
type InvitacionPendienteResponse struct {
	CodigoInvitacionProy int64     `json:"codigoInvitacionProy"`
	Token                string    `json:"token"`
	NombreProyecto       string    `json:"nombreProyecto"`
	InvitadoPor          string    `json:"invitadoPor"`
	NombreRol            string    `json:"nombreRol"`
	FechaHoraExpiracion  time.Time `json:"fechaHoraExpiracion"`
}
