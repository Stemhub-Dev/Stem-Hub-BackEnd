package service

import (
	"database/sql"
	"errors"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

var (
	ErrRolPermisoRolNoEncontrado = errors.New(
		"rol no encontrado",
	)

	ErrRolPermisoPermisosInvalidos = errors.New(
		"uno o más permisos son inválidos para el rol",
	)

	ErrRolPermisoAdministradorSinGestionRoles = errors.New(
		"el rol Administrador debe conservar el permiso GESTIONAR_ROLES",
	)
)

type RolPermisoService interface {
	ObtenerPermisosPorRol(
		codigoRol int64,
	) (*dto.RolPermisosResponse, error)

	ActualizarPermisosRol(
		codigoRol int64,
		request dto.ActualizarPermisosRolRequest,
	) (*dto.RolPermisosResponse, error)
}

type rolPermisoService struct {
	repository repository.RolPermisoRepository
}

func NewRolPermisoService(
	repository repository.RolPermisoRepository,
) RolPermisoService {

	return &rolPermisoService{
		repository: repository,
	}
}

func (s *rolPermisoService) ObtenerPermisosPorRol(
	codigoRol int64,
) (*dto.RolPermisosResponse, error) {

	response, err :=
		s.repository.ObtenerPermisosPorRol(
			codigoRol,
		)

	if err != nil {

		if errors.Is(
			err,
			sql.ErrNoRows,
		) {
			return nil,
				ErrRolPermisoRolNoEncontrado
		}

		return nil, err
	}

	return response, nil
}

func (s *rolPermisoService) ActualizarPermisosRol(
	codigoRol int64,
	request dto.ActualizarPermisosRolRequest,
) (*dto.RolPermisosResponse, error) {

	// Primero obtenemos el rol y sus permisos disponibles.
	rol, err :=
		s.repository.ObtenerPermisosPorRol(
			codigoRol,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrRolPermisoRolNoEncontrado
	}

	if err != nil {
		return nil, err
	}

	// Normalizamos para evitar IDs repetidos.
	codigosNormalizados :=
		make([]int64, 0, len(request.CodigosPermiso))

	vistos :=
		make(map[int64]struct{})

	for _, codigo := range request.CodigosPermiso {

		if codigo <= 0 {
			return nil,
				ErrRolPermisoPermisosInvalidos
		}

		if _, existe :=
			vistos[codigo]; existe {

			continue
		}

		vistos[codigo] =
			struct{}{}

		codigosNormalizados =
			append(
				codigosNormalizados,
				codigo,
			)
	}

	// Protección especial:
	// Administrador debe conservar GESTIONAR_ROLES.
	if rol.NombreRol == "Administrador" &&
		rol.AmbitoRol == "SISTEMA" {

		var codigoGestionarRoles int64

		for _, permiso := range rol.Permisos {

			if permiso.ClavePermiso ==
				"GESTIONAR_ROLES" {

				codigoGestionarRoles =
					permiso.CodigoPermiso

				break
			}
		}

		if codigoGestionarRoles != 0 {

			_, incluido :=
				vistos[codigoGestionarRoles]

			if !incluido {
				return nil,
					ErrRolPermisoAdministradorSinGestionRoles
			}
		}
	}

	response, err :=
		s.repository.ActualizarPermisosRol(
			codigoRol,
			codigosNormalizados,
		)

	if err != nil {

		if errors.Is(
			err,
			sql.ErrNoRows,
		) {
			return nil,
				ErrRolPermisoRolNoEncontrado
		}

		if errors.Is(
			err,
			repository.ErrPermisosRolInvalidos,
		) {
			return nil,
				ErrRolPermisoPermisosInvalidos
		}

		return nil, err
	}

	return response, nil
}
