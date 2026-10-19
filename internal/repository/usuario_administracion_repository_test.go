package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Integración contra PostgreSQL real (TEST_DATABASE_URL, como el resto de
// los tests de integración): el control de "último administrador" vive en
// SQL, donde un mock no detecta un nombre de rol desfasado.

func conectarBaseAdministracion(t *testing.T) *sql.DB {
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

// Crea un usuario activo con el rol Administrador del sistema y lo borra al
// terminar.
func nuevoAdministrador(t *testing.T, db *sql.DB, nombre string) int64 {
	t.Helper()

	var codigoUsuario int64
	err := db.QueryRow(
		`INSERT INTO usuario (email, idautenticacion)
		 VALUES ($1, gen_random_uuid()) RETURNING codigousuario`,
		fmt.Sprintf("%s-%d@test.local", nombre, time.Now().UnixNano()),
	).Scan(&codigoUsuario)
	if err != nil {
		t.Fatalf("crear usuario: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM usuariorol WHERE codigousuario = $1`, codigoUsuario)
		_, _ = db.Exec(`DELETE FROM usuario WHERE codigousuario = $1`, codigoUsuario)
	})

	resultado, err := db.Exec(`
		INSERT INTO usuariorol (codigousuario, codrol, ambitorol)
		SELECT $1, r.codrol, r.ambitorol
		FROM rol r
		WHERE LOWER(r.nombrerol) = LOWER($2)
		  AND r.ambitorol = 'SISTEMA'
		  AND r.fechahorabajarol IS NULL
	`, codigoUsuario, model.NombreRolAdministradorSistema)
	if err != nil {
		t.Fatalf("asignar rol: %v", err)
	}
	if filas, _ := resultado.RowsAffected(); filas != 1 {
		t.Fatalf("no existe el rol %q en la base", model.NombreRolAdministradorSistema)
	}

	return codigoUsuario
}

func contarAdministradoresActivos(t *testing.T, db *sql.DB) int {
	t.Helper()

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	// Reusa la verificación real: con un admin "previo" falso, falla solo
	// si hay cero.
	if err := verificarQuedaAdministrador(tx, true); errors.Is(err, ErrSinAdministradoresActivos) {
		return 0
	} else if err != nil {
		t.Fatal(err)
	}
	return 1
}

func TestAdministracion_QuitarRolYBajaConOtroAdministrador(t *testing.T) {
	db := conectarBaseAdministracion(t)
	repo := NewUsuarioAdministracionRepository(db)

	nuevoAdministrador(t, db, "admin-a")
	b := nuevoAdministrador(t, db, "admin-b")

	if _, err := repo.ActualizarAdministracion(b, false, true); err != nil {
		t.Fatalf("quitar rol con otro admin activo: %v", err)
	}
	if _, err := repo.ActualizarAdministracion(b, true, true); err != nil {
		t.Fatalf("volver a asignar rol: %v", err)
	}
	if _, err := repo.CambiarEstado(b, false); err != nil {
		t.Fatalf("dar de baja con otro admin activo: %v", err)
	}
}

func TestAdministracion_NoDejaAlSistemaSinAdministradores(t *testing.T) {
	db := conectarBaseAdministracion(t)
	repo := NewUsuarioAdministracionRepository(db)

	if contarAdministradoresActivos(t, db) > 0 {
		t.Skip("la base ya tiene administradores activos: no se puede armar el caso del último")
	}

	unico := nuevoAdministrador(t, db, "admin-unico")

	if _, err := repo.CambiarEstado(unico, false); !errors.Is(err, ErrSinAdministradoresActivos) {
		t.Fatalf("baja del último admin: err = %v, quería ErrSinAdministradoresActivos", err)
	}
	if _, err := repo.ActualizarAdministracion(unico, false, true); !errors.Is(err, ErrSinAdministradoresActivos) {
		t.Fatalf("quitar rol al último admin: err = %v, quería ErrSinAdministradoresActivos", err)
	}

	// El rollback dejó todo como estaba: sigue siendo admin activo.
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	sigueAdmin, err := prepararCambioAdministradores(tx, unico)
	if err != nil || !sigueAdmin {
		t.Fatalf("después del rechazo: esAdmin = %v, err = %v; quería true", sigueAdmin, err)
	}
}
