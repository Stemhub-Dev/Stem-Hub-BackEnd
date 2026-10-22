package repository

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// HU-SEG-B02 contra PostgreSQL real: sesiones revocadas e inactivas.

const inactividadPrueba = 45 * time.Minute

func nuevaSesion(t *testing.T) (SesionRepository, string, string) {
	t.Helper()
	db := conectarBaseUsuarios(t)
	idSesion := uuid.NewString()
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM sesion WHERE idsesion = $1`, idSesion)
	})
	return NewSesionRepository(db), idSesion, uuid.NewString()
}

func registrar(t *testing.T, repo SesionRepository, idSesion, idAuth string, ahora time.Time) bool {
	t.Helper()
	vigente, err := repo.RegistrarActividad(idSesion, idAuth, ahora, inactividadPrueba)
	if err != nil {
		t.Fatal(err)
	}
	return vigente
}

func TestSesion_UsoDentroDeLos45MinutosSigueVigente(t *testing.T) {
	repo, idSesion, idAuth := nuevaSesion(t)
	inicio := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)

	if !registrar(t, repo, idSesion, idAuth, inicio) {
		t.Fatal("primer uso: quería vigente")
	}
	// Cada uso corre la ventana: 44 + 44 minutos sigue sirviendo.
	if !registrar(t, repo, idSesion, idAuth, inicio.Add(44*time.Minute)) {
		t.Fatal("a los 44 minutos: quería vigente")
	}
	if !registrar(t, repo, idSesion, idAuth, inicio.Add(88*time.Minute)) {
		t.Fatal("44 minutos después del último uso: quería vigente")
	}
}

func TestSesion_45MinutosSinUsoLaVence(t *testing.T) {
	repo, idSesion, idAuth := nuevaSesion(t)
	inicio := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)

	registrar(t, repo, idSesion, idAuth, inicio)

	if registrar(t, repo, idSesion, idAuth, inicio.Add(45*time.Minute)) {
		t.Fatal("a los 45 minutos sin uso: quería vencida")
	}
	// Una vez vencida no revive, aunque se siga intentando.
	if registrar(t, repo, idSesion, idAuth, inicio.Add(46*time.Minute)) {
		t.Fatal("después de vencer: quería vencida")
	}
}

func TestSesion_RevocadaNoSirveMas(t *testing.T) {
	repo, idSesion, idAuth := nuevaSesion(t)
	ahora := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)

	registrar(t, repo, idSesion, idAuth, ahora)
	if err := repo.Revocar(idSesion, idAuth, ahora.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	if registrar(t, repo, idSesion, idAuth, ahora.Add(2*time.Minute)) {
		t.Fatal("después del logout: quería rechazada")
	}
}

func TestSesion_RevocarUnaSesionNuncaUsadaLaDejaCerrada(t *testing.T) {
	repo, idSesion, idAuth := nuevaSesion(t)
	ahora := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)

	if err := repo.Revocar(idSesion, idAuth, ahora); err != nil {
		t.Fatal(err)
	}

	if registrar(t, repo, idSesion, idAuth, ahora.Add(time.Second)) {
		t.Fatal("quería rechazada")
	}
}

func TestSesion_OtroUsuarioNoPuedeUsarNiCerrarLaSesion(t *testing.T) {
	repo, idSesion, idAuth := nuevaSesion(t)
	ahora := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
	otro := uuid.NewString()

	registrar(t, repo, idSesion, idAuth, ahora)

	if registrar(t, repo, idSesion, otro, ahora.Add(time.Minute)) {
		t.Fatal("otro usuario con el mismo session_id: quería rechazado")
	}
	if err := repo.Revocar(idSesion, otro, ahora.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !registrar(t, repo, idSesion, idAuth, ahora.Add(2*time.Minute)) {
		t.Fatal("el logout de otro usuario no debe cerrar la sesión")
	}
}

func TestSesion_UnaRequestDemoradaNoRetrocedeLaActividad(t *testing.T) {
	repo, idSesion, idAuth := nuevaSesion(t)
	inicio := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)

	registrar(t, repo, idSesion, idAuth, inicio.Add(30*time.Minute))
	registrar(t, repo, idSesion, idAuth, inicio)

	if !registrar(t, repo, idSesion, idAuth, inicio.Add(70*time.Minute)) {
		t.Fatal("quería vigente: el último uso real fue a los 30 minutos")
	}
}

func TestSesion_UsosEnParaleloDeUnaSesionNueva(t *testing.T) {
	repo, idSesion, idAuth := nuevaSesion(t)
	ahora := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	errores := make(chan error, 10)
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vigente, err := repo.RegistrarActividad(idSesion, idAuth, ahora, inactividadPrueba)
			if err == nil && !vigente {
				t.Error("uso en paralelo rechazado")
			}
			if err != nil {
				errores <- err
			}
		}()
	}
	wg.Wait()
	close(errores)
	for err := range errores {
		t.Fatal(err)
	}
}
