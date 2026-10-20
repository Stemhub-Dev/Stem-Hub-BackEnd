package service

import (
	"errors"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

// HU-SEG-F04: asociar y quitar permisos de un rol.

const codigoGestionarRoles int64 = 1

type rolPermisoRepositoryFalso struct {
	rol      dto.RolPermisosResponse
	guardado []int64
	llamado  bool
}

func (r *rolPermisoRepositoryFalso) ObtenerPermisosPorRol(int64) (*dto.RolPermisosResponse, error) {
	return &r.rol, nil
}

func (r *rolPermisoRepositoryFalso) ActualizarPermisosRol(_ int64, codigos []int64) (*dto.RolPermisosResponse, error) {
	r.llamado = true
	r.guardado = codigos
	return &r.rol, nil
}

func rolConPermisos(nombre string) *rolPermisoRepositoryFalso {
	return &rolPermisoRepositoryFalso{rol: dto.RolPermisosResponse{
		CodigoRol: 3,
		NombreRol: nombre,
		AmbitoRol: "SISTEMA",
		Permisos: []dto.PermisoRolResponse{
			{CodigoPermiso: codigoGestionarRoles, ClavePermiso: "GESTIONAR_ROLES", Asignado: true},
			{CodigoPermiso: 2, ClavePermiso: "GESTIONAR_USUARIOS", Asignado: true},
		},
	}}
}

func TestActualizarPermisosRol_AdministradorConservaGestionarRoles(t *testing.T) {
	repo := rolConPermisos(model.NombreRolAdministradorSistema)
	s := NewRolPermisoService(repo)

	_, err := s.ActualizarPermisosRol(3, dto.ActualizarPermisosRolRequest{CodigosPermiso: []int64{2}})

	if !errors.Is(err, ErrRolPermisoAdministradorSinGestionRoles) {
		t.Fatalf("err = %v, quería ErrRolPermisoAdministradorSinGestionRoles", err)
	}
	if repo.llamado {
		t.Fatal("no debería guardar el cambio")
	}
}

func TestActualizarPermisosRol_AdministradorPuedeQuitarOtrosPermisos(t *testing.T) {
	repo := rolConPermisos(model.NombreRolAdministradorSistema)
	s := NewRolPermisoService(repo)

	if _, err := s.ActualizarPermisosRol(3, dto.ActualizarPermisosRolRequest{CodigosPermiso: []int64{codigoGestionarRoles}}); err != nil {
		t.Fatalf("err = %v, quería nil", err)
	}
	if len(repo.guardado) != 1 || repo.guardado[0] != codigoGestionarRoles {
		t.Fatalf("guardado = %v, quería [%d]", repo.guardado, codigoGestionarRoles)
	}
}

func TestActualizarPermisosRol_OtroRolPuedeQuitarGestionarRoles(t *testing.T) {
	repo := rolConPermisos("Soporte")
	s := NewRolPermisoService(repo)

	if _, err := s.ActualizarPermisosRol(3, dto.ActualizarPermisosRolRequest{CodigosPermiso: []int64{}}); err != nil {
		t.Fatalf("err = %v, quería nil", err)
	}
}

func TestActualizarPermisosRol_DescartaRepetidosYRechazaInvalidos(t *testing.T) {
	repo := rolConPermisos("Soporte")
	s := NewRolPermisoService(repo)

	if _, err := s.ActualizarPermisosRol(3, dto.ActualizarPermisosRolRequest{CodigosPermiso: []int64{2, 2, 1}}); err != nil {
		t.Fatal(err)
	}
	if len(repo.guardado) != 2 {
		t.Fatalf("guardado = %v, quería sin repetidos", repo.guardado)
	}

	_, err := s.ActualizarPermisosRol(3, dto.ActualizarPermisosRolRequest{CodigosPermiso: []int64{0}})
	if !errors.Is(err, ErrRolPermisoPermisosInvalidos) {
		t.Fatalf("err = %v, quería ErrRolPermisoPermisosInvalidos", err)
	}
}
