package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"net/mail"
	"net/smtp"
	"text/template"
)

//go:embed templates/invitacion.txt
var plantillasFS embed.FS

type smtpMailer struct {
	host      string
	port      string
	user      string
	pass      string
	remitente string
	plantilla *template.Template
}

func NewSMTPMailer(host, port, user, pass, remitente string) (Mailer, error) {

	plantilla, err := template.ParseFS(plantillasFS, "templates/invitacion.txt")

	if err != nil {
		return nil, fmt.Errorf("no se pudo cargar la plantilla de invitación: %w", err)
	}

	return &smtpMailer{
		host:      host,
		port:      port,
		user:      user,
		pass:      pass,
		remitente: remitente,
		plantilla: plantilla,
	}, nil
}

func (m *smtpMailer) EnviarInvitacionProyecto(data InvitacionEmailData) error {

	var cuerpo bytes.Buffer

	if err := m.plantilla.Execute(&cuerpo, data); err != nil {
		return fmt.Errorf("no se pudo renderizar el mail de invitación: %w", err)
	}

	asunto := fmt.Sprintf("Te invitaron a colaborar en %s en StemHub", data.NombreProyecto)

	mensaje := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.remitente,
		data.EmailDestino,
		asunto,
		cuerpo.String(),
	)

	// Sin credenciales configuradas (ej. Inbucket en dev local, que no
	// implementa AUTH), se manda sin autenticar en vez de forzar un AUTH
	// que el servidor va a rechazar.
	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}

	// El envelope MAIL FROM del protocolo SMTP exige una dirección pura,
	// sin el "Display Name" — a diferencia del header From: del mensaje
	// (m.remitente), que sí lo acepta. Si el remitente no llegara a
	// parsear (config mal cargada), se cae al valor tal cual: la mayoría
	// de los servidores igual lo rechazan con el mismo error 501 que
	// ya se ve en los logs, así que no cambia el comportamiento.
	remitenteEnvelope := m.remitente
	if direccion, err := mail.ParseAddress(m.remitente); err == nil {
		remitenteEnvelope = direccion.Address
	}

	return smtp.SendMail(
		m.host+":"+m.port,
		auth,
		remitenteEnvelope,
		[]string{data.EmailDestino},
		[]byte(mensaje),
	)
}
