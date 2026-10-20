package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// La validación del nombre corre antes de tocar el repositorio, por eso
// alcanza con un service sin repositorio.
func TestRegistrarUsuario_NombreLargo(t *testing.T) {
	svc := NewUsuarioService(nil)

	_, _, _, err := svc.RegistrarUsuario(
		"auth-id",
		"ana@test.com",
		true,
		strings.Repeat("a", LargoMaximoNombreIntegrante+1),
		nil,
	)

	if err != ErrNombreIntegranteLargo {
		t.Fatalf("esperaba ErrNombreIntegranteLargo, obtuve %v", err)
	}
}

// Simula una cuenta existente con el mismo email y otro idautenticacion.
type usuarioRepositorioConEmailTomado struct {
	repository.UsuarioRepository
	existente   model.Usuario
	reasignadoA string
}

func (r *usuarioRepositorioConEmailTomado) Registrar(
	string, string, string, *string,
) (*model.Usuario, *model.Integrante, bool, error) {
	return nil, nil, false, repository.ErrEmailConOtraIdentidad
}

func (r *usuarioRepositorioConEmailTomado) ReasignarIdentidad(
	_ string,
	idAutenticacion string,
) (*model.Usuario, *model.Integrante, error) {
	r.reasignadoA = idAutenticacion
	usuario := r.existente
	usuario.IDAutenticacion = idAutenticacion
	return &usuario, &model.Integrante{CodIntegrante: 3, CodigoUsuario: usuario.CodigoUsuario}, nil
}

func TestRegistrarUsuario_EmailVerificadoRecuperaLaCuentaExistente(t *testing.T) {
	repo := &usuarioRepositorioConEmailTomado{existente: model.Usuario{CodigoUsuario: 7, IDAutenticacion: "viejo"}}
	svc := NewUsuarioService(repo)

	usuario, integrante, creado, err := svc.RegistrarUsuario("nuevo", "ana@test.com", true, "Ana", nil)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.reasignadoA != "nuevo" {
		t.Fatalf("reasignó a %q, quería la identidad nueva", repo.reasignadoA)
	}
	if usuario.CodigoUsuario != 7 || integrante.CodIntegrante != 3 {
		t.Fatalf("devolvió %d/%d, quería la cuenta existente 7/3", usuario.CodigoUsuario, integrante.CodIntegrante)
	}
	if creado {
		t.Fatal("recuperar una cuenta no es crearla")
	}
}

func TestRegistrarUsuario_CuentaRecuperadaDadaDeBajaSigueInactiva(t *testing.T) {
	baja := time.Now()
	repo := &usuarioRepositorioConEmailTomado{
		existente: model.Usuario{CodigoUsuario: 7, FechaHoraBajaUsuario: &baja},
	}
	svc := NewUsuarioService(repo)

	_, _, _, err := svc.RegistrarUsuario("nuevo", "ana@test.com", true, "Ana", nil)

	if !errors.Is(err, ErrUsuarioInactivo) {
		t.Fatalf("esperaba ErrUsuarioInactivo, obtuve %v", err)
	}
	if repo.reasignadoA != "nuevo" {
		t.Fatal("tenía que pasar la cuenta a la identidad nueva para que un admin pueda reactivarla")
	}
}

func TestRegistrarUsuario_EmailSinVerificarNoTomaLaCuentaAjena(t *testing.T) {
	repo := &usuarioRepositorioConEmailTomado{existente: model.Usuario{CodigoUsuario: 7}}
	svc := NewUsuarioService(repo)

	_, _, _, err := svc.RegistrarUsuario("nuevo", "ana@test.com", false, "Ana", nil)

	if !errors.Is(err, ErrEmailYaRegistrado) {
		t.Fatalf("esperaba ErrEmailYaRegistrado, obtuve %v", err)
	}
	if repo.reasignadoA != "" {
		t.Fatal("no debería reasignar la cuenta con un email sin verificar")
	}
}
