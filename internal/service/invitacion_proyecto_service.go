package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/mailer"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
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

	if email == "" {
		return nil, ErrInvitacionEmailInvalido
	}

	existeProyecto, err := s.proyectoRepository.ExisteProyectoActivo(codigoProyecto)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrProyectoNoEncontrado
	}

	integranteInvito, err := s.integranteRepository.BuscarPorCodigoUsuario(codigoUsuario)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvitacionNoEsOwner
	}

	if err != nil {
		return nil, err
	}

	esPropietario, err := s.proyectoRepository.EsPropietarioActivo(
		integranteInvito.CodIntegrante,
		codigoProyecto,
	)

	if err != nil {
		return nil, err
	}

	if !esPropietario {
		return nil, ErrInvitacionNoEsOwner
	}

	ambitoRol, err := s.proyectoRepository.ObtenerAmbitoRolActivo(request.CodRol)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvitacionRolNoValido
	}

	if err != nil {
		return nil, err
	}

	if ambitoRol != ambitoRolProyecto {
		return nil, ErrInvitacionRolNoValido
	}

	if err := s.validarNoEsYaIntegrante(email, codigoProyecto); err != nil {
		return nil, err
	}

	token, err := generarTokenInvitacion()

	if err != nil {
		return nil, err
	}

	expiracion := time.Now().Add(time.Duration(s.diasVencimiento) * 24 * time.Hour)

	invitacion, err := s.invitacionRepository.CrearOReemplazarPendiente(
		codigoProyecto,
		email,
		request.CodRol,
		ambitoRol,
		integranteInvito.CodIntegrante,
		token,
		expiracion,
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
		LinkAceptacion:  s.frontendBaseURL + "/invitaciones/" + token,
		DiasVencimiento: s.diasVencimiento,
	})

	if err != nil {
		log.Println("Error al enviar mail de invitación:", err)
	}

	return &dto.InvitacionResponse{
		CodigoInvitacionProy: invitacion.CodigoInvitacionProy,
		Email:                invitacion.EmailInvitado,
		CodRol:               invitacion.CodRol,
		NombreRol:            nombreRol,
		FechaHoraExpiracion:  invitacion.FechaHoraExpiracion,
		Estado:               "pendiente",
	}, nil
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

		estado := "pendiente"

		if invitacion.FechaHoraExpiracion.Before(time.Now()) {
			estado = "vencida"
		}

		respuesta = append(respuesta, dto.InvitacionResponse{
			CodigoInvitacionProy: invitacion.CodigoInvitacionProy,
			Email:                invitacion.EmailInvitado,
			CodRol:               invitacion.CodRol,
			NombreRol:            nombreRol,
			FechaHoraExpiracion:  invitacion.FechaHoraExpiracion,
			Estado:               estado,
		})
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

	if !strings.EqualFold(usuario.Email, invitacion.EmailInvitado) {
		return 0, ErrInvitacionEmailNoCoincide
	}

	integrante, err := s.integranteRepository.BuscarPorCodigoUsuario(usuario.CodigoUsuario)

	if err != nil {
		return 0, err
	}

	err = s.invitacionRepository.Aceptar(
		invitacion.CodigoInvitacionProy,
		integrante.CodIntegrante,
		invitacion.CodigoProyecto,
		invitacion.CodRol,
		invitacion.AmbitoRol,
	)

	if err != nil {
		return 0, err
	}

	return invitacion.CodigoProyecto, nil
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
