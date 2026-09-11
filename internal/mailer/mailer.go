package mailer

type InvitacionEmailData struct {
	EmailDestino    string
	NombreProyecto  string
	NombreInvitador string
	NombreRol       string
	LinkAceptacion  string
	DiasVencimiento int
}

type Mailer interface {
	EnviarInvitacionProyecto(data InvitacionEmailData) error
}
