package service

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrComentarioTextoObligatorio = errors.New(
		"el comentario no puede estar vacío",
	)

	ErrComentarioTextoMuyLargo = errors.New(
		"el comentario supera los 200 caracteres",
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
)

type ComentarioService interface {
	Crear(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		request dto.CrearComentarioRequest,
	) (*dto.CrearComentarioResponse, error)
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

	codigoComentario, err :=
		s.comentarioRepository.Crear(
			integrante.CodIntegrante,
			codigoVersion,
			request.Texto,
		)

	if err != nil {
		return nil, err
	}

	return &dto.CrearComentarioResponse{
		CodigoComentario: codigoComentario,
		Texto:            request.Texto,
		Estado:           "Pendiente",
		Autor: dto.AutorComentarioResponse{
			CodigoIntegrante: integrante.CodIntegrante,
			Nombre:           integrante.NombreIntegrante,
		},
	}, nil
}
