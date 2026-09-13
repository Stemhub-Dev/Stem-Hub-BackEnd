package service

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrComentarioTextoObligatorio = errors.New(
		"el comentario no puede estar vacío",
	)

	ErrComentarioTextoMuyLargo = errors.New(
		"el comentario supera los 200 caracteres",
	)

	ErrComentarioRangoInvalido = errors.New(
		"el rango de tiempo del comentario es inválido",
	)

	ErrComentarioProyectoNoEncontrado = errors.New(
		"el proyecto no existe",
	)

	ErrComentarioCancionNoEncontrada = errors.New(
		"la canción no existe en el proyecto",
	)

	ErrComentarioVersionNoEncontrada = errors.New(
		"la versión no existe en la canción",
	)

	ErrComentarioSinAcceso = errors.New(
		"el usuario no pertenece al proyecto",
	)

	ErrRespuestaComentarioTextoObligatorio = errors.New(
		"la respuesta no puede estar vacía",
	)

	ErrRespuestaComentarioTextoMuyLargo = errors.New(
		"la respuesta supera los 200 caracteres",
	)

	ErrComentarioNoEncontrado = errors.New(
		"el comentario no existe",
	)

	ErrComentarioNoEsPropio = errors.New(
		"no se puede modificar un comentario de otro usuario",
	)

	ErrComentarioSinPermisoEliminar = errors.New(
		"no tiene permisos para eliminar este comentario",
	)

	ErrComentarioEstadoInvalido = errors.New(
		"el estado del comentario no es válido",
	)
)

type ComentarioService interface {
	Crear(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		request dto.CrearComentarioRequest,
	) (*dto.CrearComentarioResponse, error)

	ListarPorVersion(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
	) ([]dto.ComentarioListadoResponse, error)

	Responder(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		codigoComentario int64,
		request dto.CrearRespuestaComentarioRequest,
	) (*dto.RespuestaComentarioResponse, error)

	Modificar(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		codigoComentario int64,
		request dto.ModificarComentarioRequest,
	) (*dto.ModificarComentarioResponse, error)

	Eliminar(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		codigoComentario int64,
	) error

	CambiarEstado(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		codigoComentario int64,
		request dto.CambiarEstadoComentarioRequest,
	) (*dto.CambiarEstadoComentarioResponse, error)
}

type comentarioService struct {
	comentarioRepository repository.ComentarioRepository
	proyectoRepository   repository.ProyectoRepository
	cancionRepository    repository.CancionRepository
	integranteRepository repository.IntegranteRepository
}

func NewComentarioService(
	comentarioRepository repository.ComentarioRepository,
	proyectoRepository repository.ProyectoRepository,
	cancionRepository repository.CancionRepository,
	integranteRepository repository.IntegranteRepository,
) ComentarioService {

	return &comentarioService{
		comentarioRepository: comentarioRepository,
		proyectoRepository:   proyectoRepository,
		cancionRepository:    cancionRepository,
		integranteRepository: integranteRepository,
	}
}

func (s *comentarioService) Crear(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	request dto.CrearComentarioRequest,
) (*dto.CrearComentarioResponse, error) {

	request.Texto = strings.TrimSpace(request.Texto)

	if request.Texto == "" {
		return nil, ErrComentarioTextoObligatorio
	}

	if len([]rune(request.Texto)) > 200 {
		return nil, ErrComentarioTextoMuyLargo
	}

	tieneInicio := request.TiempoInicioSegundos != nil
	tieneFin := request.TiempoFinSegundos != nil

	if tieneInicio != tieneFin {
		return nil, ErrComentarioRangoInvalido
	}

	if tieneInicio && tieneFin &&
		*request.TiempoFinSegundos < *request.TiempoInicioSegundos {
		return nil, ErrComentarioRangoInvalido
	}

	integrante, err :=
		s.validarAccesoVersion(
			codigoUsuario,
			codigoProyecto,
			codigoCancion,
			codigoVersion,
		)

	if err != nil {
		return nil, err
	}

	codigoComentario, err := s.comentarioRepository.Crear(
		integrante.CodIntegrante,
		codigoVersion,
		request.Texto,
		request.TiempoInicioSegundos,
		request.TiempoFinSegundos,
	)

	if err != nil {
		return nil, err
	}

	return &dto.CrearComentarioResponse{
		CodigoComentario:     codigoComentario,
		Texto:                request.Texto,
		Estado:               "Pendiente",
		TiempoInicioSegundos: request.TiempoInicioSegundos,
		TiempoFinSegundos:    request.TiempoFinSegundos,
		Autor: dto.AutorComentarioResponse{
			CodigoIntegrante: integrante.CodIntegrante,
			Nombre:           integrante.NombreIntegrante,
		},
	}, nil
}

func (s *comentarioService) validarAccesoVersion(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
) (*model.Integrante, error) {

	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrComentarioProyectoNoEncontrado
	}

	existeCancion, err :=
		s.cancionRepository.ExisteCancionActivaEnProyecto(
			codigoProyecto,
			codigoCancion,
		)

	if err != nil {
		return nil, err
	}

	if !existeCancion {
		return nil, ErrComentarioCancionNoEncontrada
	}

	existeVersion, err :=
		s.cancionRepository.ExisteVersionActivaEnCancion(
			codigoCancion,
			codigoVersion,
		)

	if err != nil {
		return nil, err
	}

	if !existeVersion {
		return nil, ErrComentarioVersionNoEncontrada
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrComentarioSinAcceso
	}

	if err != nil {
		return nil, err
	}

	esIntegrante, err :=
		s.proyectoRepository.EsIntegranteActivo(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !esIntegrante {
		return nil, ErrComentarioSinAcceso
	}

	return integrante, nil
}

func (s *comentarioService) ListarPorVersion(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
) ([]dto.ComentarioListadoResponse, error) {

	integrante, err :=
		s.validarAccesoVersion(
			codigoUsuario,
			codigoProyecto,
			codigoCancion,
			codigoVersion,
		)

	if err != nil {
		return nil, err
	}

	return s.comentarioRepository.ListarPorVersion(
		codigoVersion,
		integrante.CodIntegrante,
	)
}

func (s *comentarioService) Responder(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	codigoComentario int64,
	request dto.CrearRespuestaComentarioRequest,
) (*dto.RespuestaComentarioResponse, error) {

	request.Texto = strings.TrimSpace(request.Texto)

	if request.Texto == "" {
		return nil, ErrRespuestaComentarioTextoObligatorio
	}

	if len([]rune(request.Texto)) > 200 {
		return nil, ErrRespuestaComentarioTextoMuyLargo
	}

	integrante, err := s.validarAccesoVersion(
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
	)

	if err != nil {
		return nil, err
	}

	existeComentario, err :=
		s.comentarioRepository.ExisteComentarioActivoEnVersion(
			codigoComentario,
			codigoVersion,
		)

	if err != nil {
		return nil, err
	}

	if !existeComentario {
		return nil, ErrComentarioNoEncontrado
	}

	respuesta, err :=
		s.comentarioRepository.CrearRespuesta(
			codigoComentario,
			integrante.CodIntegrante,
			request.Texto,
		)

	if err != nil {
		return nil, err
	}

	respuesta.Autor = dto.AutorComentarioResponse{
		CodigoIntegrante: integrante.CodIntegrante,
		Nombre:           integrante.NombreIntegrante,
	}

	respuesta.EsPropia = true

	return respuesta, nil
}

func (s *comentarioService) Modificar(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	codigoComentario int64,
	request dto.ModificarComentarioRequest,
) (*dto.ModificarComentarioResponse, error) {

	request.Texto = strings.TrimSpace(request.Texto)

	if request.Texto == "" {
		return nil, ErrComentarioTextoObligatorio
	}

	if len([]rune(request.Texto)) > 200 {
		return nil, ErrComentarioTextoMuyLargo
	}

	integrante, err := s.validarAccesoVersion(
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
	)

	if err != nil {
		return nil, err
	}

	codigoAutor, err :=
		s.comentarioRepository.ObtenerAutorComentario(
			codigoComentario,
			codigoVersion,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrComentarioNoEncontrado
	}

	if err != nil {
		return nil, err
	}

	if codigoAutor != integrante.CodIntegrante {
		return nil, ErrComentarioNoEsPropio
	}

	err = s.comentarioRepository.Modificar(
		codigoComentario,
		request.Texto,
	)

	if err != nil {
		return nil, err
	}

	return &dto.ModificarComentarioResponse{
		CodigoComentario: codigoComentario,
		Texto:            request.Texto,
		Autor: dto.AutorComentarioResponse{
			CodigoIntegrante: integrante.CodIntegrante,
			Nombre:           integrante.NombreIntegrante,
		},
	}, nil
}

func (s *comentarioService) Eliminar(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	codigoComentario int64,
) error {

	integrante, err := s.validarAccesoVersion(
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
	)

	if err != nil {
		return err
	}

	codigoAutor, err :=
		s.comentarioRepository.ObtenerAutorComentario(
			codigoComentario,
			codigoVersion,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrComentarioNoEncontrado
	}

	if err != nil {
		return err
	}

	// El autor siempre puede eliminar su propio comentario.
	if codigoAutor != integrante.CodIntegrante {

		// Para eliminar un comentario ajeno debe tener
		// el permiso GESTIONAR_COMENTARIOS.
		puedeGestionarComentarios, err :=
			s.proyectoRepository.PuedeRealizarEnProyecto(
				integrante.CodIntegrante,
				codigoProyecto,
				"GESTIONAR_COMENTARIOS",
			)

		if err != nil {
			return err
		}

		if !puedeGestionarComentarios {
			return ErrComentarioSinPermisoEliminar
		}
	}

	eliminado, err :=
		s.comentarioRepository.DarDeBaja(
			codigoComentario,
		)

	if err != nil {
		return err
	}

	if !eliminado {
		return ErrComentarioNoEncontrado
	}

	return nil
}

func (s *comentarioService) CambiarEstado(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	codigoComentario int64,
	request dto.CambiarEstadoComentarioRequest,
) (*dto.CambiarEstadoComentarioResponse, error) {

	request.Estado = strings.TrimSpace(request.Estado)

	if request.Estado == "" {
		return nil, ErrComentarioEstadoInvalido
	}

	// Para este flujo funcional solamente admitimos
	// Pendiente y Resuelto.
	if !strings.EqualFold(request.Estado, "Pendiente") &&
		!strings.EqualFold(request.Estado, "Resuelto") {

		return nil, ErrComentarioEstadoInvalido
	}

	_, err := s.validarAccesoVersion(
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
	)

	if err != nil {
		return nil, err
	}

	existeComentario, err :=
		s.comentarioRepository.ExisteComentarioActivoEnVersion(
			codigoComentario,
			codigoVersion,
		)

	if err != nil {
		return nil, err
	}

	if !existeComentario {
		return nil, ErrComentarioNoEncontrado
	}

	existeEstado, err :=
		s.comentarioRepository.ExisteEstadoComentarioActivo(
			request.Estado,
		)

	if err != nil {
		return nil, err
	}

	if !existeEstado {
		return nil, ErrComentarioEstadoInvalido
	}

	estadoNormalizado := "Pendiente"

	if strings.EqualFold(request.Estado, "Resuelto") {
		estadoNormalizado = "Resuelto"
	}

	actualizado, err :=
		s.comentarioRepository.CambiarEstado(
			codigoComentario,
			codigoVersion,
			estadoNormalizado,
		)

	if err != nil {
		return nil, err
	}

	if !actualizado {
		return nil, ErrComentarioNoEncontrado
	}

	return &dto.CambiarEstadoComentarioResponse{
		CodigoComentario: codigoComentario,
		Estado:           estadoNormalizado,
	}, nil
}
