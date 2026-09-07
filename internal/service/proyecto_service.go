package service

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
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
)

type ProyectoService interface {
	CrearProyecto(
		codigoUsuario int64,
		request dto.CrearProyectoRequest,
	) (*dto.CrearProyectoResponse, error)

	ListarProyectos(
		codigoUsuario int64,
	) ([]dto.ProyectoListadoResponse, error)
}

type proyectoService struct {
	proyectoRepository   repository.ProyectoRepository
	integranteRepository repository.IntegranteRepository
}

func NewProyectoService(
	proyectoRepository repository.ProyectoRepository,
	integranteRepository repository.IntegranteRepository,
) ProyectoService {

	return &proyectoService{
		proyectoRepository:   proyectoRepository,
		integranteRepository: integranteRepository,
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
