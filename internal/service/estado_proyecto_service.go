package service

import (
	"errors"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrNombreEstadoProyectoRequerido = errors.New("El nombre del estado de proyecto es requerido")
	ErrEstadoProyectoDuplicado       = errors.New("El estado de proyecto ya existe")
	ErrEstadoProyectoNoEncontrado    = errors.New("El estado de proyecto no existe")
	ErrEstadoEstadoProyectoRequerido = errors.New("El campo activo es requerido")
)

type EstadoProyectoService struct {
	repository *repository.EstadoProyectoRepository
}

func NewEstadoProyectoService(
	repository *repository.EstadoProyectoRepository,
) *EstadoProyectoService {

	return &EstadoProyectoService{
		repository: repository,
	}
}

func (s *EstadoProyectoService) Listar(
	incluirInactivos bool,
) ([]dto.EstadoProyectoResponse, error) {

	estados, err :=
		s.repository.Listar(
			incluirInactivos,
		)

	if err != nil {
		return nil, err
	}

	respuesta := make(
		[]dto.EstadoProyectoResponse,
		0,
		len(estados),
	)

	for _, estado := range estados {

		respuesta = append(
			respuesta,
			dto.EstadoProyectoResponse{
				ID: estado.CodEstadoProy,

				Nombre: estado.NombreEstadoProy,

				Activo: estado.FechaHoraBajaEstadoProy == nil,
			},
		)
	}

	return respuesta, nil
}

func (s *EstadoProyectoService) Crear(
	request dto.EstadoProyectoRequest,
) (dto.EstadoProyectoResponse, error) {

	nombre :=
		strings.TrimSpace(
			request.Nombre,
		)

	if nombre == "" {
		return dto.EstadoProyectoResponse{},
			ErrNombreEstadoProyectoRequerido
	}

	existe, err :=
		s.repository.ExistePorNombre(
			nombre,
		)

	if err != nil {
		return dto.EstadoProyectoResponse{},
			err
	}

	if existe {
		return dto.EstadoProyectoResponse{},
			ErrEstadoProyectoDuplicado
	}

	estado, err :=
		s.repository.Crear(
			nombre,
		)

	if err != nil {
		return dto.EstadoProyectoResponse{},
			err
	}

	return dto.EstadoProyectoResponse{
		ID: estado.CodEstadoProy,

		Nombre: estado.NombreEstadoProy,

		Activo: estado.FechaHoraBajaEstadoProy == nil,
	}, nil
}

func (s *EstadoProyectoService) Editar(
	id int64,
	request dto.EstadoProyectoRequest,
) (dto.EstadoProyectoResponse, error) {

	nombre :=
		strings.TrimSpace(
			request.Nombre,
		)

	if nombre == "" {
		return dto.EstadoProyectoResponse{},
			ErrNombreEstadoProyectoRequerido
	}

	existe, err :=
		s.repository.ExistePorID(
			id,
		)

	if err != nil {
		return dto.EstadoProyectoResponse{},
			err
	}

	if !existe {
		return dto.EstadoProyectoResponse{},
			ErrEstadoProyectoNoEncontrado
	}

	duplicado, err :=
		s.repository.
			ExistePorNombreExcluyendoID(
				nombre,
				id,
			)

	if err != nil {
		return dto.EstadoProyectoResponse{},
			err
	}

	if duplicado {
		return dto.EstadoProyectoResponse{},
			ErrEstadoProyectoDuplicado
	}

	estado, err :=
		s.repository.Editar(
			id,
			nombre,
		)

	if err != nil {
		return dto.EstadoProyectoResponse{},
			err
	}

	return dto.EstadoProyectoResponse{
		ID: estado.CodEstadoProy,

		Nombre: estado.NombreEstadoProy,

		Activo: estado.FechaHoraBajaEstadoProy == nil,
	}, nil
}

func (s *EstadoProyectoService) CambiarEstado(
	id int64,
	request dto.CambiarEstadoProyectoRequest,
) (dto.EstadoProyectoResponse, error) {

	if request.Activo == nil {
		return dto.EstadoProyectoResponse{},
			ErrEstadoEstadoProyectoRequerido
	}

	existe, err :=
		s.repository.ExistePorID(
			id,
		)

	if err != nil {
		return dto.EstadoProyectoResponse{},
			err
	}

	if !existe {
		return dto.EstadoProyectoResponse{},
			ErrEstadoProyectoNoEncontrado
	}

	estado, err :=
		s.repository.CambiarEstado(
			id,
			*request.Activo,
		)

	if err != nil {
		return dto.EstadoProyectoResponse{},
			err
	}

	return dto.EstadoProyectoResponse{
		ID: estado.CodEstadoProy,

		Nombre: estado.NombreEstadoProy,

		Activo: estado.FechaHoraBajaEstadoProy == nil,
	}, nil
}
