package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// HU-ACA-B01: registrar es idempotente. Integración contra PostgreSQL real
// (TEST_DATABASE_URL); se saltea si no está definida.

func conectarBaseUsuarios(t *testing.T) *sql.DB {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida: se omiten los tests de integración")
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("no se pudo conectar a la base: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return db
}

func TestRegistrar_EsIdempotente(t *testing.T) {
	db := conectarBaseUsuarios(t)
	repo := NewUsuarioRepository(db)

	sufijo := time.Now().UnixNano()
	idAutenticacion := fmt.Sprintf("00000000-0000-0000-0000-%012d", sufijo%1_000_000_000_000)
	email := fmt.Sprintf("registro-%d@test.local", sufijo)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM integrante WHERE codigousuario IN (SELECT codigousuario FROM usuario WHERE email = $1)`, email)
		_, _ = db.Exec(`DELETE FROM usuario WHERE email = $1`, email)
	})

	usuario, integrante, creado, err := repo.Registrar(idAutenticacion, email, "Ana", nil)
	if err != nil {
		t.Fatalf("primer registro: %v", err)
	}
	if !creado {
		t.Fatal("el primer registro debería crear")
	}

	otraVez, integranteOtraVez, creadoOtraVez, err := repo.Registrar(idAutenticacion, email, "Otro nombre", nil)
	if err != nil {
		t.Fatalf("segundo registro: %v", err)
	}
	if creadoOtraVez {
		t.Fatal("el segundo registro no debería crear")
	}
	if otraVez.CodigoUsuario != usuario.CodigoUsuario || integranteOtraVez.CodIntegrante != integrante.CodIntegrante {
		t.Fatalf("devolvió otro usuario: %d/%d, quería %d/%d",
			otraVez.CodigoUsuario, integranteOtraVez.CodIntegrante, usuario.CodigoUsuario, integrante.CodIntegrante)
	}
	if integranteOtraVez.NombreIntegrante != "Ana" {
		t.Fatalf("nombre = %q, el registro existente no debería cambiar", integranteOtraVez.NombreIntegrante)
	}
}

func TestRegistrar_EmailConOtraIdentidadYReasignacion(t *testing.T) {
	db := conectarBaseUsuarios(t)
	repo := NewUsuarioRepository(db)

	sufijo := time.Now().UnixNano()
	idOriginal := fmt.Sprintf("00000000-0000-0000-0001-%012d", sufijo%1_000_000_000_000)
	idNuevo := fmt.Sprintf("00000000-0000-0000-0002-%012d", sufijo%1_000_000_000_000)
	email := fmt.Sprintf("reasignar-%d@test.local", sufijo)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM integrante WHERE codigousuario IN (SELECT codigousuario FROM usuario WHERE email = $1)`, email)
		_, _ = db.Exec(`DELETE FROM usuario WHERE email = $1`, email)
	})

	usuario, integrante, _, err := repo.Registrar(idOriginal, email, "Ana", nil)
	if err != nil {
		t.Fatalf("primer registro: %v", err)
	}

	if _, _, _, err = repo.Registrar(idNuevo, email, "Ana", nil); !errors.Is(err, ErrEmailConOtraIdentidad) {
		t.Fatalf("esperaba ErrEmailConOtraIdentidad, obtuve %v", err)
	}

	reasignado, integranteReasignado, err := repo.ReasignarIdentidad(email, idNuevo)
	if err != nil {
		t.Fatalf("reasignar: %v", err)
	}
	if reasignado.CodigoUsuario != usuario.CodigoUsuario || integranteReasignado.CodIntegrante != integrante.CodIntegrante {
		t.Fatal("la reasignación tiene que conservar la misma cuenta")
	}

	encontrado, err := repo.BuscarPorIDAutenticacion(idNuevo)
	if err != nil || encontrado.CodigoUsuario != usuario.CodigoUsuario {
		t.Fatalf("la identidad nueva no encuentra la cuenta: %v", err)
	}
}
