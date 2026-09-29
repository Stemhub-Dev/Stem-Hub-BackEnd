package service

import (
	"errors"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrNombreTipoProyectoRequerido = errors.New("El nombre del tipo de proyecto es requerido")
	ErrTipoProyectoDuplicado       = errors.New("El tipo de proyecto ya existe")
	ErrTipoProyectoNoEncontrado    = errors.New("El tipo de proyecto no existe")
	ErrEstadoTipoProyectoRequerido = errors.New("El campo activo es requerido")
)

type TipoProyectoService struct {
	repository *repository.TipoProyectoRepository
}

func NewTipoProyectoService(
	repository *repository.TipoProyectoRepository,
) *TipoProyectoService {

	return &TipoProyectoService{
		repository: repository,
	}
}

func (s *TipoProyectoService) Listar(
	incluirInactivos bool,
) ([]dto.TipoProyectoResponse, error) {

	tipos, err :=
		s.repository.Listar(
			incluirInactivos,
		)

	if err != nil {
		return nil, err
	}

	respuesta := make(
		[]dto.TipoProyectoResponse,
		0,
		len(tipos),
	)

	for _, tipo := range tipos {

		respuesta = append(
			respuesta,
			dto.TipoProyectoResponse{
				ID: tipo.CodTipoProy,

				Nombre: tipo.NombreTipoProy,

				Activo: tipo.FechaHoraBajaTipoProy == nil,
			},
		)
	}

	return respuesta, nil
}

func (s *TipoProyectoService) Crear(
	request dto.TipoProyectoRequest,
) (dto.TipoProyectoResponse, error) {

	nombre :=
		strings.TrimSpace(
			request.Nombre,
		)

	if nombre == "" {
		return dto.TipoProyectoResponse{},
			ErrNombreTipoProyectoRequerido
	}

	existe, err :=
		s.repository.ExistePorNombre(
			nombre,
		)

	if err != nil {
		return dto.TipoProyectoResponse{},
			err
	}

	if existe {
		return dto.TipoProyectoResponse{},
			ErrTipoProyectoDuplicado
	}

	tipo, err :=
		s.repository.Crear(
			nombre,
		)

	if err != nil {
		return dto.TipoProyectoResponse{},
			err
	}

	return dto.TipoProyectoResponse{
		ID: tipo.CodTipoProy,

		Nombre: tipo.NombreTipoProy,

		Activo: tipo.FechaHoraBajaTipoProy == nil,
	}, nil
}

func (s *TipoProyectoService) Editar(
	id int64,
	request dto.TipoProyectoRequest,
) (dto.TipoProyectoResponse, error) {

	nombre :=
		strings.TrimSpace(
			request.Nombre,
		)

	if nombre == "" {
		return dto.TipoProyectoResponse{},
			ErrNombreTipoProyectoRequerido
	}

	existe, err :=
		s.repository.ExistePorID(
			id,
		)

	if err != nil {
		return dto.TipoProyectoResponse{},
			err
	}

	if !existe {
		return dto.TipoProyectoResponse{},
			ErrTipoProyectoNoEncontrado
	}

	duplicado, err :=
		s.repository.
			ExistePorNombreExcluyendoID(
				nombre,
				id,
			)

	if err != nil {
		return dto.TipoProyectoResponse{},
			err
	}

	if duplicado {
		return dto.TipoProyectoResponse{},
			ErrTipoProyectoDuplicado
	}

	tipo, err :=
		s.repository.Editar(
			id,
			nombre,
		)

	if err != nil {
		return dto.TipoProyectoResponse{},
			err
	}

	return dto.TipoProyectoResponse{
		ID: tipo.CodTipoProy,

		Nombre: tipo.NombreTipoProy,

		Activo: tipo.FechaHoraBajaTipoProy == nil,
	}, nil
}

func (s *TipoProyectoService) CambiarEstado(
	id int64,
	request dto.CambiarEstadoTipoProyectoRequest,
) (dto.TipoProyectoResponse, error) {

	if request.Activo == nil {
		return dto.TipoProyectoResponse{},
			ErrEstadoTipoProyectoRequerido
	}

	existe, err :=
		s.repository.ExistePorID(
			id,
		)

	if err != nil {
		return dto.TipoProyectoResponse{},
			err
	}

	if !existe {
		return dto.TipoProyectoResponse{},
			ErrTipoProyectoNoEncontrado
	}

	tipo, err :=
		s.repository.CambiarEstado(
			id,
			*request.Activo,
		)

	if err != nil {
		return dto.TipoProyectoResponse{},
			err
	}

	return dto.TipoProyectoResponse{
		ID: tipo.CodTipoProy,

		Nombre: tipo.NombreTipoProy,

		Activo: tipo.FechaHoraBajaTipoProy == nil,
	}, nil
}
