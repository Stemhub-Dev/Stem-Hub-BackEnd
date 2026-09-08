package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
)

const (
	codigoEstadoProyectoInicial int64 = 1
	ambitoRolProyecto                 = "PROYECTO"
)

var (
	ErrPerfilRequerido = errors.New(
		"el usuario debe completar su perfil",
	)

	ErrTipoProyectoNoValido = errors.New(
		"el tipo de proyecto no es válido",
	)

	ErrRolProyectoNoValido = errors.New(
		"el rol no corresponde al ámbito proyecto",
	)

	ErrGenerosNoValidos = errors.New(
		"uno o más géneros no son válidos",
	)

	ErrNombreProyectoObligatorio = errors.New(
		"el nombre del proyecto es obligatorio",
	)

	ErrProyectoNoEncontrado = errors.New(
		"el proyecto no existe",
	)

	ErrProyectoSinAcceso = errors.New(
		"el usuario no pertenece al proyecto",
	)
)

type ProyectoService interface {
	CrearProyecto(
		codigoUsuario int64,
		request dto.CrearProyectoRequest,
	) (*dto.CrearProyectoResponse, error)

	ListarProyectos(
		codigoUsuario int64,
	) ([]dto.ProyectoListadoResponse, error)

	ListarColaboradores(
		codigoUsuario int64,
		codigoProyecto int64,
	) ([]dto.ColaboradorProyectoResponse, error)
}

type proyectoService struct {
	proyectoRepository   repository.ProyectoRepository
	integranteRepository repository.IntegranteRepository
	audioStorage         storage.AudioStorage
}

func NewProyectoService(
	proyectoRepository repository.ProyectoRepository,
	integranteRepository repository.IntegranteRepository,
	audioStorage storage.AudioStorage,
) ProyectoService {

	return &proyectoService{
		proyectoRepository:   proyectoRepository,
		integranteRepository: integranteRepository,
		audioStorage:         audioStorage,
	}
}

func (s *proyectoService) CrearProyecto(
	codigoUsuario int64,
	request dto.CrearProyectoRequest,
) (*dto.CrearProyectoResponse, error) {

	request.Nombre = strings.TrimSpace(request.Nombre)

	if request.Nombre == "" {
		return nil, ErrNombreProyectoObligatorio
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(codigoUsuario)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	if integrante.FechaHoraBajaIntegrante != nil {
		return nil, ErrPerfilRequerido
	}

	existeTipo, err :=
		s.proyectoRepository.ExisteTipoProyectoActivo(
			request.CodigoTipoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeTipo {
		return nil, ErrTipoProyectoNoValido
	}

	ambitoRol, err :=
		s.proyectoRepository.ObtenerAmbitoRolActivo(
			request.CodRol,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRolProyectoNoValido
	}

	if err != nil {
		return nil, err
	}

	if ambitoRol != ambitoRolProyecto {
		return nil, ErrRolProyectoNoValido
	}

	existenGeneros, err :=
		s.proyectoRepository.ExistenGeneros(
			request.CodigosGeneros,
		)

	if err != nil {
		return nil, err
	}

	if !existenGeneros {
		return nil, ErrGenerosNoValidos
	}

	codigoProyecto, err :=
		s.proyectoRepository.Crear(
			integrante.CodIntegrante,
			request.Nombre,
			request.Descripcion,
			codigoEstadoProyectoInicial,
			request.CodigoTipoProyecto,
			request.CodigosGeneros,
			request.CodRol,
			ambitoRol,
		)

	if err != nil {
		return nil, err
	}

	return &dto.CrearProyectoResponse{
		CodigoProyecto:     codigoProyecto,
		Nombre:             request.Nombre,
		Descripcion:        request.Descripcion,
		CodigoTipoProyecto: request.CodigoTipoProyecto,
		CodigosGeneros:     request.CodigosGeneros,
		CodRol:             request.CodRol,
		EsPropietario:      true,
	}, nil
}

func (s *proyectoService) ListarProyectos(
	codigoUsuario int64,
) ([]dto.ProyectoListadoResponse, error) {

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return []dto.ProyectoListadoResponse{}, nil
	}

	if err != nil {
		return nil, err
	}

	if integrante.FechaHoraBajaIntegrante != nil {
		return []dto.ProyectoListadoResponse{}, nil
	}

	return s.proyectoRepository.ListarPorIntegrante(
		integrante.CodIntegrante,
	)
}

func (s *proyectoService) ListarColaboradores(
	codigoUsuario int64,
	codigoProyecto int64,
) ([]dto.ColaboradorProyectoResponse, error) {

	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrProyectoNoEncontrado
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProyectoSinAcceso
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
		return nil, ErrProyectoSinAcceso
	}

	colaboradores, err :=
		s.proyectoRepository.ListarColaboradores(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	respuesta := make(
		[]dto.ColaboradorProyectoResponse,
		0,
		len(colaboradores),
	)

	for _, colaborador := range colaboradores {

		var avatarUrl *string

		if colaborador.AvatarObjectKey != nil {

			url, err := s.audioStorage.ObtenerURLDescarga(
				context.Background(),
				*colaborador.AvatarObjectKey,
				VigenciaURLAvatar,
			)

			if err != nil {
				return nil, err
			}

			avatarUrl = &url
		}

		respuesta = append(
			respuesta,
			dto.ColaboradorProyectoResponse{
				CodigoIntegrante: colaborador.CodIntegrante,
				Nombre:           colaborador.NombreIntegrante,
				AvatarUrl:        avatarUrl,
				CodRol:           colaborador.CodRol,
				NombreRol:        colaborador.NombreRol,
				EsPropietario:    colaborador.EsPropietario,
			},
		)
	}

	return respuesta, nil
}
