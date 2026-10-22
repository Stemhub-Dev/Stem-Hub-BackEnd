package repository

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// HU-ABM-02-01 / HU-ABM-02-02 contra PostgreSQL real: dos altas o renombres
// en paralelo con el mismo nombre no pueden dejar dos canciones iguales.

func proyectoParaNombres(t *testing.T) (CancionRepository, int64) {
	t.Helper()
	db := conectarBaseUsuarios(t)

	var proyecto int64
	if err := db.QueryRow(`
		INSERT INTO proyecto (nombreproyecto, codestadoproy, codtipoproy)
		VALUES ($1, (SELECT MIN(codestadoproy) FROM estadoproyecto), (SELECT MIN(codtipoproy) FROM tipoproyecto))
		RETURNING codigoproyecto
	`, fmt.Sprintf("nombres-paralelo-%d", time.Now().UnixNano())).Scan(&proyecto); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM cancion WHERE codigoproyecto = $1`, proyecto)
		_, _ = db.Exec(`DELETE FROM proyecto WHERE codigoproyecto = $1`, proyecto)
	})

	return NewCancionRepository(db), proyecto
}

func contarCanciones(t *testing.T, proyecto int64, nombre string) int {
	t.Helper()
	var cantidad int
	if err := conectarBaseUsuarios(t).QueryRow(
		`SELECT COUNT(*) FROM cancion WHERE codigoproyecto = $1 AND LOWER(TRIM(nombrecancion)) = LOWER($2)`,
		proyecto, nombre,
	).Scan(&cantidad); err != nil {
		t.Fatal(err)
	}
	return cantidad
}

func TestCancion_AltasEnParaleloConElMismoNombreCreanUna(t *testing.T) {
	repo, proyecto := proyectoParaNombres(t)

	const intentos = 5
	var wg sync.WaitGroup
	var mu sync.Mutex
	creadas, duplicadas := 0, 0

	for i := range intentos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Mayúsculas y espacios distintos: igual cuentan como el mismo nombre.
			nombre := "Balada"
			if i%2 == 0 {
				nombre = " balada "
			}
			tx, _, err := repo.IniciarCreacionCancion(proyecto, nombre)
			if errors.Is(err, ErrNombreCancionDuplicado) {
				mu.Lock()
				duplicadas++
				mu.Unlock()
				return
			}
			if err != nil {
				t.Error(err)
				return
			}
			// Simula la subida del audio con el lock tomado.
			time.Sleep(50 * time.Millisecond)
			if err := tx.Commit(); err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			creadas++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if creadas != 1 || duplicadas != intentos-1 {
		t.Fatalf("creadas = %d, duplicadas = %d; quería 1 y %d", creadas, duplicadas, intentos-1)
	}
	if n := contarCanciones(t, proyecto, "balada"); n != 1 {
		t.Fatalf("canciones en la base = %d, quería 1", n)
	}
}

func TestCancion_AltaQueFallaLiberaElNombre(t *testing.T) {
	repo, proyecto := proyectoParaNombres(t)

	tx, _, err := repo.IniciarCreacionCancion(proyecto, "Intro")
	if err != nil {
		t.Fatal(err)
	}
	// Falló la subida del audio: el service hace rollback.
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	tx, _, err = repo.IniciarCreacionCancion(proyecto, "Intro")
	if err != nil {
		t.Fatalf("después del rollback el nombre debería estar libre: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestCancion_RenombresEnParaleloAlMismoNombreDejanUno(t *testing.T) {
	repo, proyecto := proyectoParaNombres(t)

	const cantidad = 4
	codigos := make([]int64, cantidad)
	for i := range cantidad {
		tx, codigo, err := repo.IniciarCreacionCancion(proyecto, fmt.Sprintf("Tema %d", i))
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		codigos[i] = codigo
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	renombradas := 0
	for _, codigo := range codigos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.ActualizarNombre(proyecto, codigo, "Final")
			if errors.Is(err, ErrNombreCancionDuplicado) {
				return
			}
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			renombradas++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if renombradas != 1 {
		t.Fatalf("renombradas = %d, quería 1", renombradas)
	}
	if n := contarCanciones(t, proyecto, "final"); n != 1 {
		t.Fatalf("canciones llamadas Final = %d, quería 1", n)
	}
}

func TestCancion_RenombrarAlMismoNombreConOtrasMayusculasNoChocaConsigoMisma(t *testing.T) {
	repo, proyecto := proyectoParaNombres(t)

	tx, codigo, err := repo.IniciarCreacionCancion(proyecto, "outro")
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	if err := repo.ActualizarNombre(proyecto, codigo, "Outro"); err != nil {
		t.Fatalf("renombrar la misma canción: %v", err)
	}
}
