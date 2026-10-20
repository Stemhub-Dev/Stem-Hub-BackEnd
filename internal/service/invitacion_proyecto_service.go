package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/mailer"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"golang.org/x/net/idna"
)

var (
	ErrInvitacionNoEsOwner = errors.New(
		"solo el propietario del proyecto puede invitar",
	)

	ErrInvitacionEmailInvalido = errors.New(
		"el email es inválido",
	)

	ErrInvitacionYaEsIntegrante = errors.New(
		"el usuario ya es integrante del proyecto",
	)

	ErrInvitacionRolNoValido = errors.New(
		"el rol no corresponde al ámbito proyecto",
	)

	ErrInvitacionNoEncontrada = errors.New(
		"la invitación no existe",
	)

	ErrInvitacionVencida = errors.New(
		"la invitación venció",
	)

	ErrInvitacionYaAceptada = errors.New(
		"la invitación ya fue aceptada",
	)

	ErrInvitacionCancelada = errors.New(
		"la invitación fue cancelada",
	)

	ErrInvitacionEmailNoCoincide = errors.New(
		"el email de tu cuenta no coincide con el de la invitación",
	)
)

type InvitacionProyectoService interface {
	Invitar(
		codigoUsuario int64,
		codigoProyecto int64,
		request dto.CrearInvitacionRequest,
	) (*dto.InvitacionResponse, error)

	// CrearEnlace genera una invitación sin email, con el rol elegido, y
	// devuelve el enlace para copiar (HU-NOT-F02 "Copiar enlace").
	CrearEnlace(
		codigoUsuario int64,
		codigoProyecto int64,
		request dto.CrearEnlaceInvitacionRequest,
	) (*dto.InvitacionResponse, error)

	ListarPendientes(
		codigoUsuario int64,
		codigoProyecto int64,
	) ([]dto.InvitacionResponse, error)

	Cancelar(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoInvitacion int64,
	) error

	ObtenerDetalle(token string) (*dto.InvitacionDetalleResponse, error)

	Aceptar(
		usuario *model.Usuario,
		token string,
	) (int64, error)

	// ListarMisInvitaciones trae las invitaciones pendientes del usuario
	// autenticado, por su propio email, para la bandeja de notificaciones.
	ListarMisInvitaciones(
		usuario *model.Usuario,
	) ([]dto.InvitacionPendienteResponse, error)

	Rechazar(
		usuario *model.Usuario,
		token string,
	) error
}

type invitacionProyectoService struct {
	invitacionRepository repository.InvitacionProyectoRepository
	proyectoRepository   repository.ProyectoRepository
	integranteRepository repository.IntegranteRepository
	usuarioRepository    repository.UsuarioRepository
	rolRepository        *repository.RolRepository
	mailer               mailer.Mailer
	frontendBaseURL      string
	diasVencimiento      int
}

func NewInvitacionProyectoService(
	invitacionRepository repository.InvitacionProyectoRepository,
	proyectoRepository repository.ProyectoRepository,
	integranteRepository repository.IntegranteRepository,
	usuarioRepository repository.UsuarioRepository,
	rolRepository *repository.RolRepository,
	mailer mailer.Mailer,
	frontendBaseURL string,
	diasVencimiento int,
) InvitacionProyectoService {

	return &invitacionProyectoService{
		invitacionRepository: invitacionRepository,
		proyectoRepository:   proyectoRepository,
		integranteRepository: integranteRepository,
		usuarioRepository:    usuarioRepository,
		rolRepository:        rolRepository,
		mailer:               mailer,
		frontendBaseURL:      frontendBaseURL,
		diasVencimiento:      diasVencimiento,
	}
}

func generarTokenInvitacion() (string, error) {

	valorAleatorio := make([]byte, 32)

	if _, err := rand.Read(valorAleatorio); err != nil {
		return "", err
	}

	return hex.EncodeToString(valorAleatorio), nil
}

func (s *invitacionProyectoService) Invitar(
	codigoUsuario int64,
	codigoProyecto int64,
	request dto.CrearInvitacionRequest,
) (*dto.InvitacionResponse, error) {

	email := strings.TrimSpace(strings.ToLower(request.Email))

	if !esEmailValido(email) {
		return nil, ErrInvitacionEmailInvalido
	}

	integranteInvito, ambitoRol, err := s.validarPuedeInvitar(
		codigoUsuario,
		codigoProyecto,
		request.CodRol,
	)

	if err != nil {
		return nil, err
	}

	if err := s.validarNoEsYaIntegrante(email, codigoProyecto); err != nil {
		return nil, err
	}

	token, err := generarTokenInvitacion()

	if err != nil {
		return nil, err
	}

	invitacion, err := s.invitacionRepository.CrearOReemplazarPendiente(
		codigoProyecto,
		email,
		request.CodRol,
		ambitoRol,
		integranteInvito.CodIntegrante,
		token,
		s.calcularExpiracion(),
	)

	if err != nil {
		return nil, err
	}

	nombreRol, err := s.rolRepository.ObtenerNombreRol(request.CodRol)

	if err != nil {
		return nil, err
	}

	nombreProyecto, err := s.proyectoRepository.ObtenerNombreProyecto(codigoProyecto)

	if err != nil {
		return nil, err
	}

	err = s.mailer.EnviarInvitacionProyecto(mailer.InvitacionEmailData{
		EmailDestino:    email,
		NombreProyecto:  nombreProyecto,
		NombreInvitador: integranteInvito.NombreIntegrante,
		NombreRol:       nombreRol,
		LinkAceptacion:  s.enlaceInvitacion(token),
		DiasVencimiento: s.diasVencimiento,
	})

	// La invitación queda creada aunque falle el mail: el front avisa y
	// ofrece copiar el enlace para compartirlo a mano.
	emailEnviado := err == nil

	if !emailEnviado {
		log.Println("Error al enviar mail de invitación:", err)
	}

	respuesta := s.armarRespuesta(*invitacion, nombreRol)
	respuesta.EmailEnviado = &emailEnviado

	return &respuesta, nil
}

func (s *invitacionProyectoService) CrearEnlace(
	codigoUsuario int64,
	codigoProyecto int64,
	request dto.CrearEnlaceInvitacionRequest,
) (*dto.InvitacionResponse, error) {

	integranteInvito, ambitoRol, err := s.validarPuedeInvitar(
		codigoUsuario,
		codigoProyecto,
		request.CodRol,
	)

	if err != nil {
		return nil, err
	}

	token, err := generarTokenInvitacion()

	if err != nil {
		return nil, err
	}

	invitacion, err := s.invitacionRepository.CrearEnlace(
		codigoProyecto,
		request.CodRol,
		ambitoRol,
		integranteInvito.CodIntegrante,
		token,
		s.calcularExpiracion(),
	)

	if err != nil {
		return nil, err
	}

	nombreRol, err := s.rolRepository.ObtenerNombreRol(request.CodRol)

	if err != nil {
		return nil, err
	}

	respuesta := s.armarRespuesta(*invitacion, nombreRol)

	return &respuesta, nil
}

// validarPuedeInvitar valida lo común a invitar por email y por enlace:
// proyecto activo, quien invita es propietario y el rol es de proyecto.
func (s *invitacionProyectoService) validarPuedeInvitar(
	codigoUsuario int64,
	codigoProyecto int64,
	codRol int64,
) (*model.Integrante, string, error) {

	existeProyecto, err := s.proyectoRepository.ExisteProyectoActivo(codigoProyecto)

	if err != nil {
		return nil, "", err
	}

	if !existeProyecto {
		return nil, "", ErrProyectoNoEncontrado
	}

	integranteInvito, err := s.integranteRepository.BuscarPorCodigoUsuario(codigoUsuario)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrInvitacionNoEsOwner
	}

	if err != nil {
		return nil, "", err
	}

	esPropietario, err := s.proyectoRepository.EsPropietarioActivo(
		integranteInvito.CodIntegrante,
		codigoProyecto,
	)

	if err != nil {
		return nil, "", err
	}

	if !esPropietario {
		return nil, "", ErrInvitacionNoEsOwner
	}

	ambitoRol, err := s.proyectoRepository.ObtenerAmbitoRolActivo(codRol)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrInvitacionRolNoValido
	}

	if err != nil {
		return nil, "", err
	}

	if ambitoRol != ambitoRolProyecto {
		return nil, "", ErrInvitacionRolNoValido
	}

	return integranteInvito, ambitoRol, nil
}

func (s *invitacionProyectoService) calcularExpiracion() time.Time {
	return time.Now().Add(time.Duration(s.diasVencimiento) * 24 * time.Hour)
}

func (s *invitacionProyectoService) enlaceInvitacion(token string) string {
	return s.frontendBaseURL + "/invitaciones/" + token
}

func (s *invitacionProyectoService) armarRespuesta(
	invitacion model.InvitacionProyecto,
	nombreRol string,
) dto.InvitacionResponse {

	estado := "pendiente"

	if invitacion.FechaHoraExpiracion.Before(time.Now()) {
		estado = "vencida"
	}

	return dto.InvitacionResponse{
		CodigoInvitacionProy: invitacion.CodigoInvitacionProy,
		Email:                invitacion.EmailInvitado,
		PorEnlace:            invitacion.EmailInvitado == "",
		CodRol:               invitacion.CodRol,
		NombreRol:            nombreRol,
		FechaHoraExpiracion:  invitacion.FechaHoraExpiracion,
		Estado:               estado,
		Enlace:               s.enlaceInvitacion(invitacion.TokenInvitacion),
	}
}

// validarNoEsYaIntegrante chequea si el email invitado ya pertenece a un
// usuario/integrante que forma parte activa del proyecto. Si el email no
// tiene cuenta en StemHub todavía, no hay nada que validar (la invitación
// queda pendiente por email, sin usuario asociado).
func (s *invitacionProyectoService) validarNoEsYaIntegrante(
	email string,
	codigoProyecto int64,
) error {

	usuarioInvitado, err := s.usuarioRepository.BuscarPorEmail(email)

	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	integranteInvitado, err := s.integranteRepository.BuscarPorCodigoUsuario(
		usuarioInvitado.CodigoUsuario,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return err
	}

	yaEsIntegrante, err := s.proyectoRepository.EsIntegranteActivo(
		integranteInvitado.CodIntegrante,
		codigoProyecto,
	)

	if err != nil {
		return err
	}

	if yaEsIntegrante {
		return ErrInvitacionYaEsIntegrante
	}

	return nil
}

func (s *invitacionProyectoService) ListarPendientes(
	codigoUsuario int64,
	codigoProyecto int64,
) ([]dto.InvitacionResponse, error) {

	if err := s.validarEsOwner(codigoUsuario, codigoProyecto); err != nil {
		return nil, err
	}

	invitaciones, err := s.invitacionRepository.ListarPendientesPorProyecto(codigoProyecto)

	if err != nil {
		return nil, err
	}

	respuesta := make([]dto.InvitacionResponse, 0, len(invitaciones))

	for _, invitacion := range invitaciones {

		nombreRol, err := s.rolRepository.ObtenerNombreRol(invitacion.CodRol)

		if err != nil {
			return nil, err
		}

		respuesta = append(respuesta, s.armarRespuesta(invitacion, nombreRol))
	}

	return respuesta, nil
}

func (s *invitacionProyectoService) Cancelar(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoInvitacion int64,
) error {

	if err := s.validarEsOwner(codigoUsuario, codigoProyecto); err != nil {
		return err
	}

	invitacion, err := s.invitacionRepository.BuscarPendientePorID(
		codigoInvitacion,
		codigoProyecto,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvitacionNoEncontrada
	}

	if err != nil {
		return err
	}

	return s.invitacionRepository.MarcarCancelada(invitacion.CodigoInvitacionProy)
}

// validarEsOwner resuelve el integrante del usuario autenticado y confirma
// que es propietario activo del proyecto — mismo criterio usado en
// Invitar, reusado acá para listar/cancelar.
func (s *invitacionProyectoService) validarEsOwner(
	codigoUsuario int64,
	codigoProyecto int64,
) error {

	existeProyecto, err := s.proyectoRepository.ExisteProyectoActivo(codigoProyecto)

	if err != nil {
		return err
	}

	if !existeProyecto {
		return ErrProyectoNoEncontrado
	}

	integrante, err := s.integranteRepository.BuscarPorCodigoUsuario(codigoUsuario)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvitacionNoEsOwner
	}

	if err != nil {
		return err
	}

	esPropietario, err := s.proyectoRepository.EsPropietarioActivo(
		integrante.CodIntegrante,
		codigoProyecto,
	)

	if err != nil {
		return err
	}

	if !esPropietario {
		return ErrInvitacionNoEsOwner
	}

	return nil
}

func (s *invitacionProyectoService) ObtenerDetalle(
	token string,
) (*dto.InvitacionDetalleResponse, error) {

	detalle, err := s.invitacionRepository.BuscarDetallePorToken(token)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvitacionNoEncontrada
	}

	if err != nil {
		return nil, err
	}

	return &dto.InvitacionDetalleResponse{
		Email:          detalle.EmailInvitado,
		PorEnlace:      detalle.EmailInvitado == "",
		NombreProyecto: detalle.NombreProyecto,
		InvitadoPor:    detalle.NombreIntegranteInvito,
		NombreRol:      detalle.NombreRol,
		Vencida:        detalle.Vencida,
		YaAceptada:     detalle.YaAceptada,
		Cancelada:      detalle.Cancelada,
	}, nil
}

func (s *invitacionProyectoService) Aceptar(
	usuario *model.Usuario,
	token string,
) (int64, error) {

	invitacion, err := s.buscarInvitacionValidaPorToken(token)

	if err != nil {
		return 0, err
	}

	if !emailPuedeUsarInvitacion(usuario.Email, invitacion) {
		return 0, ErrInvitacionEmailNoCoincide
	}

	// El proyecto debe seguir activo antes de aceptar.
	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			invitacion.CodigoProyecto,
		)

	if err != nil {
		return 0, err
	}

	if !existeProyecto {
		return 0, ErrProyectoNoEncontrado
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			usuario.CodigoUsuario,
		)

	if err != nil {
		return 0, err
	}

	// Con un enlace lo puede abrir alguien que ya está en el proyecto.
	yaEsIntegrante, err := s.proyectoRepository.EsIntegranteActivo(
		integrante.CodIntegrante,
		invitacion.CodigoProyecto,
	)

	if err != nil {
		return 0, err
	}

	if yaEsIntegrante {
		return 0, ErrInvitacionYaEsIntegrante
	}

	err = s.invitacionRepository.Aceptar(
		invitacion.CodigoInvitacionProy,
		integrante.CodIntegrante,
		invitacion.CodigoProyecto,
		invitacion.CodRol,
		invitacion.AmbitoRol,
	)

	// Entre la validación y el UPDATE otra request la aceptó, o se canceló
	// o venció: se vuelve a leer para responder el motivo real.
	if errors.Is(err, repository.ErrInvitacionNoDisponible) {
		if _, errEstado := s.buscarInvitacionValidaPorToken(token); errEstado != nil {
			return 0, errEstado
		}
		return 0, ErrInvitacionYaAceptada
	}

	if errors.Is(err, repository.ErrYaEsIntegranteDelProyecto) {
		return 0, ErrInvitacionYaEsIntegrante
	}

	if err != nil {
		return 0, err
	}

	return invitacion.CodigoProyecto, nil
}

func (s *invitacionProyectoService) ListarMisInvitaciones(
	usuario *model.Usuario,
) ([]dto.InvitacionPendienteResponse, error) {

	invitaciones, err := s.invitacionRepository.ListarPendientesPorEmail(usuario.Email)

	if err != nil {
		return nil, err
	}

	respuesta := make([]dto.InvitacionPendienteResponse, 0, len(invitaciones))

	for _, invitacion := range invitaciones {
		respuesta = append(respuesta, dto.InvitacionPendienteResponse{
			CodigoInvitacionProy: invitacion.CodigoInvitacionProy,
			Token:                invitacion.TokenInvitacion,
			NombreProyecto:       invitacion.NombreProyecto,
			InvitadoPor:          invitacion.NombreIntegranteInvito,
			NombreRol:            invitacion.NombreRol,
			FechaHoraExpiracion:  invitacion.FechaHoraExpiracion,
		})
	}

	return respuesta, nil
}

func (s *invitacionProyectoService) Rechazar(
	usuario *model.Usuario,
	token string,
) error {

	invitacion, err := s.buscarInvitacionValidaPorToken(token)

	if err != nil {
		return err
	}

	// Un enlace es compartido: que alguien lo rechace no lo invalida para
	// el resto (cancelarlo es del propietario).
	if invitacion.EmailInvitado == "" {
		return nil
	}

	if !emailPuedeUsarInvitacion(usuario.Email, invitacion) {
		return ErrInvitacionEmailNoCoincide
	}

	return s.invitacionRepository.MarcarCancelada(invitacion.CodigoInvitacionProy)
}

// buscarInvitacionValidaPorToken resuelve la invitación por token y valida
// su estado (existe, no cancelada, no aceptada, no vencida) sin todavía
// mirar el usuario que la está aceptando.
func (s *invitacionProyectoService) buscarInvitacionValidaPorToken(
	token string,
) (*model.InvitacionProyecto, error) {

	invitacion, err := s.invitacionRepository.BuscarPorToken(token)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvitacionNoEncontrada
	}

	if err != nil {
		return nil, err
	}

	if invitacion.FechaHoraBajaInvitacionProy != nil {
		return nil, ErrInvitacionCancelada
	}

	if invitacion.FechaHoraAceptacion != nil {
		return nil, ErrInvitacionYaAceptada
	}

	if invitacion.FechaHoraExpiracion.Before(time.Now()) {
		return nil, ErrInvitacionVencida
	}

	return invitacion, nil
}

// emailPuedeUsarInvitacion: una invitación por email solo la usa esa
// cuenta; una por enlace (sin email), cualquiera que tenga el enlace.
func emailPuedeUsarInvitacion(email string, invitacion *model.InvitacionProyecto) bool {
	return invitacion.EmailInvitado == "" ||
		strings.EqualFold(email, invitacion.EmailInvitado)
}

// largoMaximoEmail: emailinvitado es VARCHAR(254) (RFC 5321).
const largoMaximoEmail = 254

const largoMaximoParteLocal = 64

// esEmailValido acepta solo una dirección suelta ("a@b.com"): sin nombre
// ni <> (que net/mail permite), con un punto en el dominio y que entre en
// la columna.
func esEmailValido(email string) bool {
	if len(email) > largoMaximoEmail {
		return false
	}
	// RFC 5321: la parte local tiene como máximo 64 bytes.
	if arroba := strings.LastIndex(email, "@"); arroba > largoMaximoParteLocal {
		return false
	}
	direccion, err := mail.ParseAddress(email)
	if err != nil || direccion.Address != email {
		return false
	}
	return dominioValido(email[strings.LastIndex(email, "@")+1:])
}

// dominioValido aplica las reglas de nombres de host: al menos dos
// etiquetas de 1 a 63 caracteres, solo letras, dígitos y guiones, sin guion
// al principio ni al final, y la última solo letras.
func dominioValido(dominio string) bool {
	etiquetas := strings.Split(dominio, ".")
	if len(etiquetas) < 2 {
		return false
	}
	for _, etiqueta := range etiquetas {
		if len(etiqueta) == 0 || len(etiqueta) > 63 ||
			strings.HasPrefix(etiqueta, "-") || strings.HasSuffix(etiqueta, "-") {
			return false
		}
		for _, c := range etiqueta {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return tldValido(etiquetas[len(etiquetas)-1])
}

// tldValido acepta un TLD de letras ASCII o uno internacionalizado en
// Punycode (ej. xn--p1ai = .рф) que decodifique a letras y marcas
// combinantes (ej. xn--h2brj9c = .भारत, con la vocal ा).
func tldValido(tld string) bool {
	texto := tld
	if strings.HasPrefix(strings.ToLower(tld), "xn--") {
		unicodeTLD, err := idna.Lookup.ToUnicode(tld)
		if err != nil || unicodeTLD == tld {
			return false
		}
		texto = unicodeTLD
	} else {
		for _, c := range tld {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
				return false
			}
		}
		return true
	}
	for _, c := range texto {
		if !unicode.IsLetter(c) && !unicode.IsMark(c) {
			return false
		}
	}
	return true
}
