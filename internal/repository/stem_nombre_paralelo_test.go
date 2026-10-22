package repository

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// HU-ABM-04-01 contra PostgreSQL real: dos altas en paralelo del mismo stem
// (p. ej. un doble clic en "Agregar stem") pasan las dos el chequeo de
// ExisteNombreEnVersion; la segunda choca con el índice único y tiene que
// volver como duplicado, no como error interno.

func versionParaStems(t *testing.T) (StemRepository, int64, int64, int64) {
	t.Helper()
	db := conectarBaseUsuarios(t)
	sufijo := time.Now().UnixNano()

	var proyecto, cancion, version, usuario int64
	debe := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}

	debe(db.QueryRow(`
		INSERT INTO proyecto (nombreproyecto, codestadoproy, codtipoproy)
		VALUES ($1, (SELECT MIN(codestadoproy) FROM estadoproyecto), (SELECT MIN(codtipoproy) FROM tipoproyecto))
		RETURNING codigoproyecto
	`, fmt.Sprintf("stems-paralelo-%d", sufijo)).Scan(&proyecto))
	debe(db.QueryRow(
		`INSERT INTO cancion (codigoproyecto, nombrecancion) VALUES ($1, 'Tema') RETURNING codigocancion`, proyecto,
	).Scan(&cancion))
	debe(db.QueryRow(
		`INSERT INTO cancionversion (codigocancion, numeroversion, formatoarchivocancionver) VALUES ($1, 1, 'mp3') RETURNING codigocancionversion`, cancion,
	).Scan(&version))
	debe(db.QueryRow(
		`INSERT INTO usuario (email, idautenticacion) VALUES ($1, gen_random_uuid()) RETURNING codigousuario`,
		fmt.Sprintf("stems-paralelo-%d@test.local", sufijo),
	).Scan(&usuario))
	_, err := db.Exec(`INSERT INTO integrante (codigousuario, nombreintegrante) VALUES ($1, 'Productora')`, usuario)
	debe(err)

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM stem WHERE codigocancionversion = $1`, version)
		_, _ = db.Exec(`DELETE FROM categoriastem WHERE codigoproyecto = $1`, proyecto)
		_, _ = db.Exec(`DELETE FROM cancionversion WHERE codigocancionversion = $1`, version)
		_, _ = db.Exec(`DELETE FROM cancion WHERE codigocancion = $1`, cancion)
		_, _ = db.Exec(`DELETE FROM proyecto WHERE codigoproyecto = $1`, proyecto)
		_, _ = db.Exec(`DELETE FROM integrante WHERE codigousuario = $1`, usuario)
		_, _ = db.Exec(`DELETE FROM usuario WHERE codigousuario = $1`, usuario)
	})

	return NewStemRepository(db), proyecto, version, usuario
}

func TestStem_AltasEnParaleloConElMismoNombreDevuelvenDuplicado(t *testing.T) {
	repo, proyecto, version, usuario := versionParaStems(t)
	categoria := categoriaBajo(t)

	const intentos = 4
	var wg sync.WaitGroup
	var mu sync.Mutex
	creados, duplicados := 0, 0
	for i := range intentos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Crear(usuario, proyecto, version, "Bajo", categoria, nil, ArchivoStemGuardado{
				URL: fmt.Sprintf("stems/%d.wav", i), Formato: "wav", NombreOriginal: "bajo.wav",
			})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				creados++
			case errors.Is(err, ErrNombreStemDuplicado):
				duplicados++
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	if creados != 1 || duplicados != intentos-1 {
		t.Fatalf("creados = %d, duplicados = %d; quería 1 y %d", creados, duplicados, intentos-1)
	}
}

func TestStem_CategoriaNuevaEnParaleloDevuelveCategoriaDuplicada(t *testing.T) {
	repo, proyecto, version, usuario := versionParaStems(t)

	var wg sync.WaitGroup
	errs := make([]error, 3)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nueva := "Teclados"
			_, errs[i] = repo.Crear(usuario, proyecto, version, fmt.Sprintf("Teclado %d", i), 0, &nueva, ArchivoStemGuardado{
				URL: fmt.Sprintf("stems/k%d.wav", i), Formato: "wav", NombreOriginal: "k.wav",
			})
		}()
	}
	wg.Wait()

	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case !errors.Is(err, ErrCategoriaStemDuplicada):
			t.Fatalf("err = %v, quería ErrCategoriaStemDuplicada", err)
		}
	}
	if ok != 1 {
		t.Fatalf("altas exitosas = %d, quería 1", ok)
	}
}

func categoriaBajo(t *testing.T) int64 {
	t.Helper()
	var categoria int64
	if err := conectarBaseUsuarios(t).QueryRow(
		`SELECT codcategoriastem FROM categoriastem WHERE codigoproyecto IS NULL AND nombrecategoriastem = 'Bajo'`,
	).Scan(&categoria); err != nil {
		t.Fatal(err)
	}
	return categoria
}
