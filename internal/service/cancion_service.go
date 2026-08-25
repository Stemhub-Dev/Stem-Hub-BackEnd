package service

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrCancionNombreObligatorio = errors.New(
		"el nombre de la canción es obligatorio",
	)

	ErrCancionPistaObligatoria = errors.New(
		"la pista inicial es obligatoria",
	)

	ErrCancionProyectoNoEncontrado = errors.New(
		"el proyecto no existe",
	)

	ErrCancionSinPermiso = errors.New(
		"el usuario no puede crear canciones en este proyecto",
	)

	ErrCancionNombreDuplicado = errors.New(
		"ya existe una canción con ese nombre en el proyecto",
	)

	ErrCancionPerfilRequerido = errors.New(
		"el usuario no posee perfil",
	)

	ErrVersionPistaObligatoria = errors.New(
		"la pista de la nueva versión es obligatoria",
	)

	ErrVersionCancionNoEncontrada = errors.New(
		"la canción no existe en el proyecto",
	)

	ErrVersionSinPermiso = errors.New(
		"el usuario no puede crear versiones en este proyecto",
	)
)

type CancionService interface {
	Crear(
		codigoUsuario int64,
		codigoProyecto int64,
		request dto.CrearCancionRequest,
	) (*dto.CrearCancionResponse, error)

	CrearVersion(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		request dto.CrearVersionCancionRequest,
	) (*dto.CrearVersionCancionResponse, error)
}

type cancionService struct {
	cancionRepository    repository.CancionRepository
	proyectoRepository   repository.ProyectoRepository
	integranteRepository repository.IntegranteRepository
}

func NewCancionService(
	cancionRepository repository.CancionRepository,
	proyectoRepository repository.ProyectoRepository,
	integranteRepository repository.IntegranteRepository,
) CancionService {

	return &cancionService{
		cancionRepository:    cancionRepository,
		proyectoRepository:   proyectoRepository,
		integranteRepository: integranteRepository,
	}
}

func (s *cancionService) Crear(
	codigoUsuario int64,
	codigoProyecto int64,
	request dto.CrearCancionRequest,
) (*dto.CrearCancionResponse, error) {

	request.Nombre = strings.TrimSpace(request.Nombre)

	if request.Nombre == "" {
		return nil, ErrCancionNombreObligatorio
	}

	if request.URLVersionWAV == nil &&
		request.URLVersionMP3 == nil {

		return nil, ErrCancionPistaObligatoria
	}

	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrCancionProyectoNoEncontrado
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	puedeGestionar, err :=
		s.proyectoRepository.PuedeGestionarCanciones(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !puedeGestionar {
		return nil, ErrCancionSinPermiso
	}

	existeNombre, err :=
		s.cancionRepository.ExisteNombreEnProyecto(
			codigoProyecto,
			request.Nombre,
		)

	if err != nil {
		return nil, err
	}

	if existeNombre {
		return nil, ErrCancionNombreDuplicado
	}

	codigoCancion,
		codigoVersion,
		err :=
		s.cancionRepository.CrearConVersionInicial(
			codigoProyecto,
			request.Nombre,
			request.URLVersionWAV,
			request.URLVersionMP3,
		)

	if err != nil {
		return nil, err
	}

	return &dto.CrearCancionResponse{
		CodigoCancion:        codigoCancion,
		NombreCancion:        request.Nombre,
		CodigoCancionVersion: codigoVersion,
		NumeroVersion:        1,
		EtiquetaVersion:      "v1.0.0",
	}, nil
}

func (s *cancionService) CrearVersion(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	request dto.CrearVersionCancionRequest,
) (*dto.CrearVersionCancionResponse, error) {

	if request.URLVersionWAV == nil &&
		request.URLVersionMP3 == nil {

		return nil, ErrVersionPistaObligatoria
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
		return nil, ErrVersionCancionNoEncontrada
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	puedeCrearVersion, err :=
		s.proyectoRepository.PuedeRealizarEnProyecto(
			integrante.CodIntegrante,
			codigoProyecto,
			"GESTIONAR_VERSIONES",
		)

	if err != nil {
		return nil, err
	}

	if !puedeCrearVersion {
		return nil, ErrVersionSinPermiso
	}

	codigoVersion,
		numeroVersion,
		err :=
		s.cancionRepository.CrearVersion(
			codigoCancion,
			request.URLVersionWAV,
			request.URLVersionMP3,
		)

	if err != nil {
		return nil, err
	}

	etiqueta := "v1." +
		strconv.Itoa(numeroVersion-1) +
		".0"

	return &dto.CrearVersionCancionResponse{
		CodigoCancionVersion: codigoVersion,
		CodigoCancion:        codigoCancion,
		NumeroVersion:        numeroVersion,
		EtiquetaVersion:      etiqueta,
	}, nil
}
