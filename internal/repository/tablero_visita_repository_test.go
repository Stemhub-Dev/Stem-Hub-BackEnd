package repository

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// RegistrarVisita (Inicio, "Mientras no estabas") contra PostgreSQL real.

func nuevoUsuarioVisita(t *testing.T) (TableroRepository, int64) {
	t.Helper()
	db := conectarBaseUsuarios(t)
	var usuario int64
	if err := db.QueryRow(
		`INSERT INTO usuario (email, idautenticacion) VALUES ($1, gen_random_uuid()) RETURNING codigousuario`,
		fmt.Sprintf("visita-%d@test.local", time.Now().UnixNano()),
	).Scan(&usuario); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM usuario WHERE codigousuario = $1`, usuario)
	})
	return NewTableroRepository(db), usuario
}

func TestRegistrarVisita_DevuelveLaVisitaAnterior(t *testing.T) {
	repo, usuario := nuevoUsuarioVisita(t)
	primera := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	segunda := primera.Add(24 * time.Hour)

	anterior, err := repo.RegistrarVisita(usuario, primera)
	if err != nil || anterior != nil {
		t.Fatalf("primera visita: anterior = %v, err = %v; quería nil", anterior, err)
	}

	anterior, err = repo.RegistrarVisita(usuario, segunda)
	if err != nil || anterior == nil || !anterior.Equal(primera) {
		t.Fatalf("segunda visita: anterior = %v, err = %v; quería %v", anterior, err, primera)
	}
}

func TestRegistrarVisita_UnaRequestDemoradaNoRetrocedeLaMarca(t *testing.T) {
	repo, usuario := nuevoUsuarioVisita(t)
	reciente := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	vieja := reciente.Add(-time.Hour)

	if _, err := repo.RegistrarVisita(usuario, reciente); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.RegistrarVisita(usuario, vieja); err != nil {
		t.Fatal(err)
	}

	anterior, err := repo.RegistrarVisita(usuario, reciente.Add(time.Hour))
	if err != nil || anterior == nil || !anterior.Equal(reciente) {
		t.Fatalf("anterior = %v, err = %v; quería %v", anterior, err, reciente)
	}
}

// Dos visitas a la vez se serializan: una ve la marca previa y la otra, la
// que dejó la primera; nunca las dos la misma.
func TestRegistrarVisita_VisitasSimultaneasNoLeenLoMismo(t *testing.T) {
	repo, usuario := nuevoUsuarioVisita(t)
	inicial := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	if _, err := repo.RegistrarVisita(usuario, inicial); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	resultados := make([]*time.Time, 2)
	for i := range resultados {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			anterior, err := repo.RegistrarVisita(usuario, inicial.Add(time.Duration(i+1)*time.Hour))
			if err != nil {
				t.Error(err)
			}
			resultados[i] = anterior
		}(i)
	}
	wg.Wait()

	if resultados[0] == nil || resultados[1] == nil || resultados[0].Equal(*resultados[1]) {
		t.Fatalf("anteriores = %v y %v: tenían que ser distintos", resultados[0], resultados[1])
	}
}
