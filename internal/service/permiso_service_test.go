package service

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// HU-SEG-F06: ABM de permisos.

type permisoRepositoryFalso struct {
	repository.PermisoRepository

	permisos    map[int64]*model.Permiso
	asignado    map[int64]bool // asociado hoy a un rol
	tuvoRoles   map[int64]bool // tuvo alguna vez una asociación
	creado      bool
	editado     bool
	nuevoEstado *bool
}

func nuevoRepoPermisos() *permisoRepositoryFalso {
	return &permisoRepositoryFalso{
		permisos: map[int64]*model.Permiso{
			1: {CodigoPermiso: 1, NombrePermiso: "Gestionar roles", ClavePermiso: "GESTIONAR_ROLES", AmbitoPermiso: "SISTEMA"},
			2: {CodigoPermiso: 2, NombrePermiso: "Ver reportes", ClavePermiso: "VER_REPORTES", AmbitoPermiso: "SISTEMA"},
		},
		asignado:  map[int64]bool{1: true},
		tuvoRoles: map[int64]bool{1: true},
	}
}

func (r *permisoRepositoryFalso) ObtenerPorID(codigo int64) (*model.Permiso, error) {
	p, ok := r.permisos[codigo]
	if !ok {
		return nil, sql.ErrNoRows
	}
	copia := *p
	return &copia, nil
}

func (r *permisoRepositoryFalso) existe(excluir int64, igual func(*model.Permiso) bool) (bool, error) {
	for codigo, p := range r.permisos {
		if codigo != excluir && igual(p) {
			return true, nil
		}
	}
	return false, nil
}

func (r *permisoRepositoryFalso) ExistePorNombre(nombre string) (bool, error) {
	return r.ExistePorNombreExcluyendoID(nombre, 0)
}

func (r *permisoRepositoryFalso) ExistePorClave(clave string) (bool, error) {
	return r.ExistePorClaveExcluyendoID(clave, 0)
}

func (r *permisoRepositoryFalso) ExistePorNombreExcluyendoID(nombre string, excluir int64) (bool, error) {
	return r.existe(excluir, func(p *model.Permiso) bool { return strings.EqualFold(p.NombrePermiso, nombre) })
}

func (r *permisoRepositoryFalso) ExistePorClaveExcluyendoID(clave string, excluir int64) (bool, error) {
	return r.existe(excluir, func(p *model.Permiso) bool { return strings.EqualFold(p.ClavePermiso, clave) })
}

func (r *permisoRepositoryFalso) TieneAsignacionesActivas(codigo int64) (bool, error) {
	return r.asignado[codigo], nil
}

func (r *permisoRepositoryFalso) TieneAsignaciones(codigo int64) (bool, error) {
	return r.tuvoRoles[codigo], nil
}

func (r *permisoRepositoryFalso) Crear(nombre string, descripcion *string, ambito, clave string) (*model.Permiso, error) {
	r.creado = true
	return &model.Permiso{CodigoPermiso: 9, NombrePermiso: nombre, DescripcionPermiso: descripcion, AmbitoPermiso: ambito, ClavePermiso: clave}, nil
}

func (r *permisoRepositoryFalso) Editar(codigo int64, nombre string, descripcion *string, ambito, clave string) (*model.Permiso, error) {
	r.editado = true
	return &model.Permiso{CodigoPermiso: codigo, NombrePermiso: nombre, DescripcionPermiso: descripcion, AmbitoPermiso: ambito, ClavePermiso: clave}, nil
}

func (r *permisoRepositoryFalso) CambiarEstado(codigo int64, activo bool) (*model.Permiso, error) {
	r.nuevoEstado = &activo
	p := *r.permisos[codigo]
	if !activo {
		ahora := time.Now()
		p.FechaHoraBajaPermiso = &ahora
	}
	return &p, nil
}

func pedido(nombre, clave, ambito string) dto.PermisoRequest {
	return dto.PermisoRequest{NombrePermiso: nombre, ClavePermiso: clave, AmbitoPermiso: ambito}
}

func TestCrearPermiso_Validaciones(t *testing.T) {
	casos := []struct {
		nombre string
		pedido dto.PermisoRequest
		quiero error
	}{
		{"sin nombre", pedido("  ", "NUEVO", "SISTEMA"), ErrNombrePermisoRequerido},
		{"sin clave", pedido("Nuevo", " ", "SISTEMA"), ErrClavePermisoRequerida},
		{"clave con guión", pedido("Nuevo", "NUEVO-PERMISO", "SISTEMA"), ErrClavePermisoInvalida},
		{"clave con espacio", pedido("Nuevo", "NUEVO PERMISO", "SISTEMA"), ErrClavePermisoInvalida},
		{"ámbito inválido", pedido("Nuevo", "NUEVO", "GLOBAL"), ErrAmbitoPermisoInvalido},
		{"nombre repetido", pedido("ver REPORTES", "NUEVO", "SISTEMA"), ErrNombrePermisoDuplicado},
		{"clave repetida", pedido("Nuevo", "ver_reportes", "SISTEMA"), ErrClavePermisoDuplicada},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := nuevoRepoPermisos()

			_, err := NewPermisoService(repo).Crear(caso.pedido)

			if !errors.Is(err, caso.quiero) {
				t.Fatalf("err = %v, quería %v", err, caso.quiero)
			}
			if repo.creado {
				t.Fatal("no debería crear")
			}
		})
	}
}

func TestCrearPermiso_ValidoNormalizaClaveYAmbito(t *testing.T) {
	repo := nuevoRepoPermisos()

	permiso, err := NewPermisoService(repo).Crear(pedido(" Gestionar usuarios ", "gestionar_usuarios", "sistema"))

	if err != nil {
		t.Fatal(err)
	}
	if permiso.ClavePermiso != "GESTIONAR_USUARIOS" || permiso.AmbitoPermiso != "SISTEMA" || permiso.NombrePermiso != "Gestionar usuarios" {
		t.Fatalf("permiso = %+v", permiso)
	}
	if !permiso.Activo {
		t.Fatal("un permiso nuevo debería quedar activo")
	}
}

func TestEditarPermiso_AsociadoNoCambiaClaveNiAmbito(t *testing.T) {
	repo := nuevoRepoPermisos()
	s := NewPermisoService(repo)

	if _, err := s.Editar(1, pedido("Gestionar roles", "ADMINISTRAR_ROLES", "SISTEMA")); !errors.Is(err, ErrClavePermisoNoEditable) {
		t.Fatalf("cambiar clave: err = %v, quería ErrClavePermisoNoEditable", err)
	}
	if _, err := s.Editar(1, pedido("Gestionar roles", "GESTIONAR_ROLES", "PROYECTO")); !errors.Is(err, ErrAmbitoPermisoNoEditable) {
		t.Fatalf("cambiar ámbito: err = %v, quería ErrAmbitoPermisoNoEditable", err)
	}
	if repo.editado {
		t.Fatal("no debería editar")
	}

	// Nombre y descripción sí se pueden cambiar.
	if _, err := s.Editar(1, pedido("Administrar roles", "GESTIONAR_ROLES", "SISTEMA")); err != nil {
		t.Fatalf("cambiar nombre: %v", err)
	}
}

func TestEditarPermiso_LibreCambiaClaveYAmbito(t *testing.T) {
	repo := nuevoRepoPermisos()

	if _, err := NewPermisoService(repo).Editar(2, pedido("Ver reportes", "VER_INFORMES", "PROYECTO")); err != nil {
		t.Fatal(err)
	}
	if !repo.editado {
		t.Fatal("debería editar")
	}
}

func TestEditarPermiso_NoChocaConsigoMismoPeroSiConOtro(t *testing.T) {
	s := NewPermisoService(nuevoRepoPermisos())

	if _, err := s.Editar(2, pedido("Ver reportes", "VER_REPORTES", "SISTEMA")); err != nil {
		t.Fatalf("guardar sin cambios: %v", err)
	}
	if _, err := s.Editar(2, pedido("Gestionar roles", "VER_REPORTES", "SISTEMA")); !errors.Is(err, ErrNombrePermisoDuplicado) {
		t.Fatalf("err = %v, quería ErrNombrePermisoDuplicado", err)
	}
}

func TestCambiarEstadoPermiso(t *testing.T) {
	desactivar := false
	activar := true

	t.Run("asociado a un rol no se desactiva", func(t *testing.T) {
		repo := nuevoRepoPermisos()

		_, err := NewPermisoService(repo).CambiarEstado(1, dto.CambiarEstadoPermisoRequest{Activo: &desactivar})

		if !errors.Is(err, ErrPermisoEnUso) {
			t.Fatalf("err = %v, quería ErrPermisoEnUso", err)
		}
		if repo.nuevoEstado != nil {
			t.Fatal("no debería cambiar el estado")
		}
	})

	t.Run("libre se desactiva (baja lógica)", func(t *testing.T) {
		repo := nuevoRepoPermisos()

		permiso, err := NewPermisoService(repo).CambiarEstado(2, dto.CambiarEstadoPermisoRequest{Activo: &desactivar})

		if err != nil {
			t.Fatal(err)
		}
		if permiso.Activo {
			t.Fatal("debería quedar inactivo")
		}
	})

	t.Run("se reactiva aunque esté asociado", func(t *testing.T) {
		repo := nuevoRepoPermisos()

		permiso, err := NewPermisoService(repo).CambiarEstado(1, dto.CambiarEstadoPermisoRequest{Activo: &activar})

		if err != nil || !permiso.Activo {
			t.Fatalf("permiso = %+v, err = %v; quería activo", permiso, err)
		}
	})

	t.Run("inexistente", func(t *testing.T) {
		_, err := NewPermisoService(nuevoRepoPermisos()).CambiarEstado(99, dto.CambiarEstadoPermisoRequest{Activo: &activar})

		if !errors.Is(err, ErrPermisoNoEncontrado) {
			t.Fatalf("err = %v, quería ErrPermisoNoEncontrado", err)
		}
	})
}
