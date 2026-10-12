package service

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrNombrePermisoRequerido = errors.New(
		"el nombre del permiso es requerido",
	)

	ErrClavePermisoRequerida = errors.New(
		"la clave técnica del permiso es requerida",
	)

	ErrClavePermisoInvalida = errors.New(
		"la clave técnica solo puede contener letras mayúsculas, números y guiones bajos",
	)

	ErrAmbitoPermisoRequerido = errors.New(
		"el ámbito del permiso es requerido",
	)

	ErrAmbitoPermisoInvalido = errors.New(
		"el ámbito del permiso debe ser SISTEMA o PROYECTO",
	)

	ErrNombrePermisoDuplicado = errors.New(
		"ya existe un permiso con ese nombre",
	)

	ErrClavePermisoDuplicada = errors.New(
		"ya existe un permiso con esa clave técnica",
	)

	ErrPermisoNoEncontrado = errors.New(
		"el permiso no existe",
	)

	ErrEstadoPermisoRequerido = errors.New(
		"el campo activo es requerido",
	)

	ErrPermisoEnUso = errors.New(
		"no se puede desactivar el permiso porque está asociado a uno o más roles",
	)

	ErrAmbitoPermisoNoEditable = errors.New(
		"no se puede cambiar el ámbito del permiso porque ya tiene asociaciones con roles",
	)

	ErrClavePermisoNoEditable = errors.New(
		"no se puede cambiar la clave técnica del permiso mientras esté asociado a un rol",
	)
)

var patronClavePermiso = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

type PermisoService interface {
	TienePermiso(
		codigoUsuario int64,
		clavePermiso string,
	) (bool, error)

	// Lo mantenemos por compatibilidad con el handler actual.
	ListarActivos() (
		[]dto.PermisoResponse,
		error,
	)

	Listar(
		incluirInactivos bool,
	) ([]dto.PermisoResponse, error)

	Crear(
		request dto.PermisoRequest,
	) (dto.PermisoResponse, error)

	Editar(
		codigoPermiso int64,
		request dto.PermisoRequest,
	) (dto.PermisoResponse, error)

	CambiarEstado(
		codigoPermiso int64,
		request dto.CambiarEstadoPermisoRequest,
	) (dto.PermisoResponse, error)
}

type permisoService struct {
	repository repository.PermisoRepository
}

func NewPermisoService(
	repository repository.PermisoRepository,
) PermisoService {

	return &permisoService{
		repository: repository,
	}
}

func (s *permisoService) TienePermiso(
	codigoUsuario int64,
	clavePermiso string,
) (bool, error) {

	return s.repository.TienePermiso(
		codigoUsuario,
		clavePermiso,
	)
}

func (s *permisoService) ListarActivos() (
	[]dto.PermisoResponse,
	error,
) {

	return s.Listar(false)
}

func (s *permisoService) Listar(
	incluirInactivos bool,
) ([]dto.PermisoResponse, error) {

	permisos, err :=
		s.repository.Listar(
			incluirInactivos,
		)

	if err != nil {
		return nil, err
	}

	respuesta := make(
		[]dto.PermisoResponse,
		0,
		len(permisos),
	)

	for _, permiso := range permisos {

		respuesta = append(
			respuesta,
			mapearPermisoResponse(
				permiso,
			),
		)
	}

	return respuesta, nil
}

func (s *permisoService) Crear(
	request dto.PermisoRequest,
) (dto.PermisoResponse, error) {

	nombre :=
		strings.TrimSpace(
			request.NombrePermiso,
		)

	clave :=
		strings.ToUpper(
			strings.TrimSpace(
				request.ClavePermiso,
			),
		)

	ambito :=
		strings.ToUpper(
			strings.TrimSpace(
				request.AmbitoPermiso,
			),
		)

	descripcion :=
		normalizarDescripcionPermiso(
			request.DescripcionPermiso,
		)

	if nombre == "" {
		return dto.PermisoResponse{},
			ErrNombrePermisoRequerido
	}

	if clave == "" {
		return dto.PermisoResponse{},
			ErrClavePermisoRequerida
	}

	if !patronClavePermiso.MatchString(
		clave,
	) {
		return dto.PermisoResponse{},
			ErrClavePermisoInvalida
	}

	if ambito == "" {
		return dto.PermisoResponse{},
			ErrAmbitoPermisoRequerido
	}

	if !ambitoPermisoValido(
		ambito,
	) {
		return dto.PermisoResponse{},
			ErrAmbitoPermisoInvalido
	}

	existeNombre, err :=
		s.repository.ExistePorNombre(
			nombre,
		)

	if err != nil {
		return dto.PermisoResponse{},
			err
	}

	if existeNombre {
		return dto.PermisoResponse{},
			ErrNombrePermisoDuplicado
	}

	existeClave, err :=
		s.repository.ExistePorClave(
			clave,
		)

	if err != nil {
		return dto.PermisoResponse{},
			err
	}

	if existeClave {
		return dto.PermisoResponse{},
			ErrClavePermisoDuplicada
	}

	permiso, err :=
		s.repository.Crear(
			nombre,
			descripcion,
			ambito,
			clave,
		)

	if err != nil {
		return dto.PermisoResponse{},
			err
	}

	return mapearPermisoResponse(
		*permiso,
	), nil
}

func (s *permisoService) Editar(
	codigoPermiso int64,
	request dto.PermisoRequest,
) (dto.PermisoResponse, error) {

	nombre :=
		strings.TrimSpace(
			request.NombrePermiso,
		)

	clave :=
		strings.ToUpper(
			strings.TrimSpace(
				request.ClavePermiso,
			),
		)

	ambito :=
		strings.ToUpper(
			strings.TrimSpace(
				request.AmbitoPermiso,
			),
		)

	descripcion :=
		normalizarDescripcionPermiso(
			request.DescripcionPermiso,
		)

	if nombre == "" {
		return dto.PermisoResponse{},
			ErrNombrePermisoRequerido
	}

	if clave == "" {
		return dto.PermisoResponse{},
			ErrClavePermisoRequerida
	}

	if !patronClavePermiso.MatchString(
		clave,
	) {
		return dto.PermisoResponse{},
			ErrClavePermisoInvalida
	}

	if ambito == "" {
		return dto.PermisoResponse{},
			ErrAmbitoPermisoRequerido
	}

	if !ambitoPermisoValido(
		ambito,
	) {
		return dto.PermisoResponse{},
			ErrAmbitoPermisoInvalido
	}

	actual, err :=
		s.repository.ObtenerPorID(
			codigoPermiso,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return dto.PermisoResponse{},
			ErrPermisoNoEncontrado
	}

	if err != nil {
		return dto.PermisoResponse{},
			err
	}

	existeNombre, err :=
		s.repository.
			ExistePorNombreExcluyendoID(
				nombre,
				codigoPermiso,
			)

	if err != nil {
		return dto.PermisoResponse{},
			err
	}

	if existeNombre {
		return dto.PermisoResponse{},
			ErrNombrePermisoDuplicado
	}

	existeClave, err :=
		s.repository.
			ExistePorClaveExcluyendoID(
				clave,
				codigoPermiso,
			)

	if err != nil {
		return dto.PermisoResponse{},
			err
	}

	if existeClave {
		return dto.PermisoResponse{},
			ErrClavePermisoDuplicada
	}

	// El ámbito no puede modificarse si el permiso
	// ya tuvo alguna asociación con un rol.
	if !strings.EqualFold(
		actual.AmbitoPermiso,
		ambito,
	) {

		tieneAsignaciones, err :=
			s.repository.TieneAsignaciones(
				codigoPermiso,
			)

		if err != nil {
			return dto.PermisoResponse{},
				err
		}

		if tieneAsignaciones {
			return dto.PermisoResponse{},
				ErrAmbitoPermisoNoEditable
		}
	}

	// La clave técnica no puede modificarse mientras
	// el permiso esté actualmente asociado a un rol.
	if !strings.EqualFold(
		actual.ClavePermiso,
		clave,
	) {

		tieneAsignacionesActivas, err :=
			s.repository.
				TieneAsignacionesActivas(
					codigoPermiso,
				)

		if err != nil {
			return dto.PermisoResponse{},
				err
		}

		if tieneAsignacionesActivas {
			return dto.PermisoResponse{},
				ErrClavePermisoNoEditable
		}
	}

	permiso, err :=
		s.repository.Editar(
			codigoPermiso,
			nombre,
			descripcion,
			ambito,
			clave,
		)

	if err != nil {
		return dto.PermisoResponse{},
			err
	}

	return mapearPermisoResponse(
		*permiso,
	), nil
}

func (s *permisoService) CambiarEstado(
	codigoPermiso int64,
	request dto.CambiarEstadoPermisoRequest,
) (dto.PermisoResponse, error) {

	if request.Activo == nil {
		return dto.PermisoResponse{},
			ErrEstadoPermisoRequerido
	}

	_, err :=
		s.repository.ObtenerPorID(
			codigoPermiso,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return dto.PermisoResponse{},
			ErrPermisoNoEncontrado
	}

	if err != nil {
		return dto.PermisoResponse{},
			err
	}

	// Para desactivar debe estar libre de asociaciones activas.
	if !*request.Activo {

		enUso, err :=
			s.repository.
				TieneAsignacionesActivas(
					codigoPermiso,
				)

		if err != nil {
			return dto.PermisoResponse{},
				err
		}

		if enUso {
			return dto.PermisoResponse{},
				ErrPermisoEnUso
		}
	}

	permiso, err :=
		s.repository.CambiarEstado(
			codigoPermiso,
			*request.Activo,
		)

	if err != nil {
		return dto.PermisoResponse{},
			err
	}

	return mapearPermisoResponse(
		*permiso,
	), nil
}

func ambitoPermisoValido(
	ambito string,
) bool {

	return ambito == "SISTEMA" ||
		ambito == "PROYECTO"
}

func normalizarDescripcionPermiso(
	descripcion *string,
) *string {

	if descripcion == nil {
		return nil
	}

	valor :=
		strings.TrimSpace(
			*descripcion,
		)

	if valor == "" {
		return nil
	}

	return &valor
}

func mapearPermisoResponse(
	permiso model.Permiso,
) dto.PermisoResponse {

	return dto.PermisoResponse{
		CodigoPermiso: permiso.CodigoPermiso,

		NombrePermiso: permiso.NombrePermiso,

		DescripcionPermiso: permiso.DescripcionPermiso,

		AmbitoPermiso: permiso.AmbitoPermiso,

		ClavePermiso: permiso.ClavePermiso,

		Activo: permiso.FechaHoraBajaPermiso == nil,
	}
}
