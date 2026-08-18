package service

import (
	"errors"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrNombreGeneroMusicalRequerido = errors.New("El nombre del género musical es requerido")
	ErrGeneroMusicalDuplicado       = errors.New("El género musical ya existe")
	ErrGeneroMusicalNoEncontrado    = errors.New("El género musical no existe")
	ErrEstadoGeneroMusicalRequerido = errors.New("El campo activo es requerido")
)

type GeneroMusicalService struct {
	repository *repository.GeneroMusicalRepository
}

func NewGeneroMusicalService(repository *repository.GeneroMusicalRepository) *GeneroMusicalService {
	return &GeneroMusicalService{
		repository: repository,
	}
}

func (s *GeneroMusicalService) Listar() (
	[]dto.GeneroMusicalResponse, error) {
	generos, err := s.repository.Listar()
	if err != nil {
		return nil, err
	}

	response := make([]dto.GeneroMusicalResponse, 0, len(generos))

	for _, genero := range generos {
		response = append(response, dto.GeneroMusicalResponse{
			ID:     genero.CodigoGeneroProy,
			Nombre: genero.NombreGeneroProy,
			Activo: genero.FechaHoraBajaGeneroProy == nil,
		})
	}

	return response, nil
}

func (s *GeneroMusicalService) Crear(
	request dto.GeneroMusicalRequest) (dto.GeneroMusicalResponse, error) {
	nombre := strings.TrimSpace(request.Nombre)

	if nombre == "" {
		return dto.GeneroMusicalResponse{}, ErrNombreGeneroMusicalRequerido
	}

	existe, err := s.repository.ExistePorNombre(nombre)
	if err != nil {
		return dto.GeneroMusicalResponse{}, err
	}

	if existe {
		return dto.GeneroMusicalResponse{}, ErrGeneroMusicalDuplicado
	}

	genero, err := s.repository.Crear(nombre)
	if err != nil {
		return dto.GeneroMusicalResponse{}, err
	}

	response := dto.GeneroMusicalResponse{
		ID:     genero.CodigoGeneroProy,
		Nombre: genero.NombreGeneroProy,
		Activo: genero.FechaHoraBajaGeneroProy == nil,
	}

	return response, nil

}

func (s *GeneroMusicalService) Editar(
	id int64,
	request dto.GeneroMusicalRequest,
) (dto.GeneroMusicalResponse, error) {

	nombre := strings.TrimSpace(request.Nombre)

	if nombre == "" {
		return dto.GeneroMusicalResponse{}, ErrNombreGeneroMusicalRequerido
	}

	existe, err := s.repository.ExistePorID(id)
	if err != nil {
		return dto.GeneroMusicalResponse{}, err
	}

	if !existe {
		return dto.GeneroMusicalResponse{}, ErrGeneroMusicalNoEncontrado
	}

	duplicado, err := s.repository.ExistePorNombreExcluyendoID(nombre, id)
	if err != nil {
		return dto.GeneroMusicalResponse{}, err
	}

	if duplicado {
		return dto.GeneroMusicalResponse{}, ErrGeneroMusicalDuplicado
	}

	genero, err := s.repository.Editar(id, nombre)
	if err != nil {
		return dto.GeneroMusicalResponse{}, err
	}

	response := dto.GeneroMusicalResponse{
		ID:     genero.CodigoGeneroProy,
		Nombre: genero.NombreGeneroProy,
		Activo: genero.FechaHoraBajaGeneroProy == nil,
	}

	return response, nil
}

func (s *GeneroMusicalService) CambiarEstado(
	id int64,
	request dto.CambiarEstadoGeneroMusicalRequest,
) (dto.GeneroMusicalResponse, error) {

	if request.Activo == nil {
		return dto.GeneroMusicalResponse{}, ErrEstadoGeneroMusicalRequerido
	}

	existe, err := s.repository.ExistePorID(id)
	if err != nil {
		return dto.GeneroMusicalResponse{}, err
	}

	if !existe {
		return dto.GeneroMusicalResponse{}, ErrGeneroMusicalNoEncontrado
	}

	genero, err := s.repository.CambiarEstado(id, *request.Activo)
	if err != nil {
		return dto.GeneroMusicalResponse{}, err
	}

	response := dto.GeneroMusicalResponse{
		ID:     genero.CodigoGeneroProy,
		Nombre: genero.NombreGeneroProy,
		Activo: genero.FechaHoraBajaGeneroProy == nil,
	}

	return response, nil
}
