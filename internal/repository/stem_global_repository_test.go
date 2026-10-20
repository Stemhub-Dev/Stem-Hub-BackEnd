package repository

import (
	"database/sql"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// HU-BUQ-B03: GET /stems busca, filtra y pagina los stems de los proyectos
// donde el usuario es Productor. Integración contra PostgreSQL real
// (TEST_DATABASE_URL); se saltea si no está definida.

type escenarioStemsGlobales struct {
	db          *sql.DB
	integrante  int64
	catBateria  int64
	catBajo     int64
	proyectoMio int64
}

func prepararStemsGlobales(t *testing.T) escenarioStemsGlobales {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida: se omiten los tests de integración")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	debe := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("carga del escenario: %v", err)
		}
	}
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())

	var (
		e                    escenarioStemsGlobales
		usuario              int64
		proyectos, canciones []int64
		codEstado, codTipo   int64
		rolProductor         int64
		rolMusico            int64
	)
	e.db = db

	t.Cleanup(func() {
		for _, c := range canciones {
			_, _ = db.Exec(`DELETE FROM stem WHERE codigocancionversion IN (SELECT codigocancionversion FROM cancionversion WHERE codigocancion = $1)`, c)
			_, _ = db.Exec(`DELETE FROM cancionversion WHERE codigocancion = $1`, c)
			_, _ = db.Exec(`DELETE FROM cancion WHERE codigocancion = $1`, c)
		}
		for _, p := range proyectos {
			_, _ = db.Exec(`DELETE FROM integranteproyecto WHERE codigoproyecto = $1`, p)
			_, _ = db.Exec(`DELETE FROM proyecto WHERE codigoproyecto = $1`, p)
		}
		_, _ = db.Exec(`DELETE FROM integrante WHERE codigousuario = $1`, usuario)
		_, _ = db.Exec(`DELETE FROM usuario WHERE codigousuario = $1`, usuario)
	})

	debe(db.QueryRow(`SELECT MIN(codestadoproy) FROM estadoproyecto`).Scan(&codEstado))
	debe(db.QueryRow(`SELECT MIN(codtipoproy) FROM tipoproyecto`).Scan(&codTipo))
	debe(db.QueryRow(`SELECT codrol FROM rol WHERE LOWER(nombrerol) = LOWER($1)`, model.NombreRolProductor).Scan(&rolProductor))
	debe(db.QueryRow(`SELECT codrol FROM rol WHERE LOWER(nombrerol) = LOWER($1)`, model.NombreRolMusicoArtista).Scan(&rolMusico))
	debe(db.QueryRow(`SELECT codcategoriastem FROM categoriastem WHERE codigoproyecto IS NULL AND nombrecategoriastem = 'Batería'`).Scan(&e.catBateria))
	debe(db.QueryRow(`SELECT codcategoriastem FROM categoriastem WHERE codigoproyecto IS NULL AND nombrecategoriastem = 'Bajo'`).Scan(&e.catBajo))

	debe(db.QueryRow(
		`INSERT INTO usuario (email, idautenticacion) VALUES ($1, gen_random_uuid()) RETURNING codigousuario`,
		"stems-globales-"+sufijo+"@test.local",
	).Scan(&usuario))
	debe(db.QueryRow(
		`INSERT INTO integrante (codigousuario, nombreintegrante) VALUES ($1, 'Productora') RETURNING codintegrante`,
		usuario,
	).Scan(&e.integrante))

	// Proyecto con el rol dado y una canción con una versión; devuelve la versión.
	proyecto := func(nombre string, rol int64) (int64, int64) {
		var p, c, v int64
		debe(db.QueryRow(
			`INSERT INTO proyecto (nombreproyecto, codestadoproy, codtipoproy) VALUES ($1, $2, $3) RETURNING codigoproyecto`,
			nombre+" "+sufijo, codEstado, codTipo,
		).Scan(&p))
		proyectos = append(proyectos, p)
		_, err := db.Exec(
			`INSERT INTO integranteproyecto (codintegrante, codigoproyecto, codrol) VALUES ($1, $2, $3)`,
			e.integrante, p, rol,
		)
		debe(err)
		debe(db.QueryRow(
			`INSERT INTO cancion (codigoproyecto, nombrecancion) VALUES ($1, 'Tema') RETURNING codigocancion`, p,
		).Scan(&c))
		canciones = append(canciones, c)
		debe(db.QueryRow(
			`INSERT INTO cancionversion (codigocancion, numeroversion, formatoarchivocancionver) VALUES ($1, 1, 'mp3') RETURNING codigocancionversion`, c,
		).Scan(&v))
		return p, v
	}

	stem := func(version int64, nombre string, categoria int64, baja bool) {
		var fechaBaja any
		if baja {
			fechaBaja = time.Now()
		}
		_, err := db.Exec(
			`INSERT INTO stem (codigocancionversion, nombrestem, codcategoriastem, urlarchivostem, formatoarchivostem, nombrearchivostem, fechahorabajastem)
			 VALUES ($1, $2, $3, 'stems/x.wav', 'wav', $5, $4)`,
			version, nombre, categoria, fechaBaja, nombre+".wav",
		)
		debe(err)
	}

	var versionMia, versionAjena int64
	e.proyectoMio, versionMia = proyecto("Productora de", rolProductor)
	_, versionAjena = proyecto("Músico en", rolMusico)

	stem(versionMia, "Bajo eléctrico", e.catBajo, false)
	stem(versionMia, "Batería acústica", e.catBateria, false)
	stem(versionMia, "Batería 100%", e.catBateria, false)
	stem(versionMia, "Bajo borrado", e.catBajo, true)
	stem(versionAjena, "Bajo ajeno", e.catBajo, false)

	return e
}

func (e escenarioStemsGlobales) listar(t *testing.T, filtro dto.ListarStemsGlobalFiltro) (int, []string) {
	t.Helper()

	repo := NewStemRepository(e.db)
	if filtro.Pagina == 0 {
		filtro.Pagina, filtro.TamanoPagina = 1, 50
	}

	total, err := repo.ContarGlobalPorProductor(e.integrante, filtro)
	if err != nil {
		t.Fatal(err)
	}
	stems, err := repo.ListarGlobalPorProductor(e.integrante, filtro, (filtro.Pagina-1)*filtro.TamanoPagina)
	if err != nil {
		t.Fatal(err)
	}

	nombres := make([]string, 0, len(stems))
	for _, s := range stems {
		if s.CodigoProyecto != e.proyectoMio {
			t.Errorf("stem %q de otro proyecto (%d)", s.Nombre, s.CodigoProyecto)
		}
		nombres = append(nombres, s.Nombre)
	}
	sort.Strings(nombres)
	return total, nombres
}

func TestStemsGlobales_SoloProyectosDondeEsProductor(t *testing.T) {
	e := prepararStemsGlobales(t)

	total, nombres := e.listar(t, dto.ListarStemsGlobalFiltro{})

	if total != 3 || fmt.Sprint(nombres) != "[Bajo eléctrico Batería 100% Batería acústica]" {
		t.Fatalf("total = %d, stems = %v; quería los 3 activos del proyecto donde es Productor", total, nombres)
	}
}

func TestStemsGlobales_TraeOrigenYDatosDelStem(t *testing.T) {
	e := prepararStemsGlobales(t)

	stems, err := NewStemRepository(e.db).ListarGlobalPorProductor(e.integrante, dto.ListarStemsGlobalFiltro{
		Busqueda: "eléctrico", Pagina: 1, TamanoPagina: 10,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(stems) != 1 {
		t.Fatalf("stems = %+v", stems)
	}
	s := stems[0]
	if s.NombreCategoria != "Bajo" || s.NombreArchivo != "Bajo eléctrico.wav" || s.FormatoArchivo != "wav" ||
		s.NombreProyecto == "" || s.NombreCancion != "Tema" || s.NumeroVersion != 1 || s.CodigoCancionVersion == 0 {
		t.Fatalf("stem = %+v", s)
	}
}

func TestStemsGlobales_BuscaPorNombre(t *testing.T) {
	e := prepararStemsGlobales(t)

	total, nombres := e.listar(t, dto.ListarStemsGlobalFiltro{Busqueda: "BATERÍA"})
	if total != 2 || len(nombres) != 2 {
		t.Fatalf("q=BATERÍA: total = %d, stems = %v", total, nombres)
	}

	// Los comodines de LIKE se buscan literalmente.
	total, nombres = e.listar(t, dto.ListarStemsGlobalFiltro{Busqueda: "100%"})
	if total != 1 || fmt.Sprint(nombres) != "[Batería 100%]" {
		t.Fatalf("q=100%%: total = %d, stems = %v", total, nombres)
	}
	if total, _ := e.listar(t, dto.ListarStemsGlobalFiltro{Busqueda: "%"}); total != 1 {
		t.Fatalf("q=%%: total = %d, quería 1", total)
	}
}

func TestStemsGlobales_FiltraPorCategoria(t *testing.T) {
	e := prepararStemsGlobales(t)

	total, nombres := e.listar(t, dto.ListarStemsGlobalFiltro{Categoria: &e.catBajo})

	if total != 1 || fmt.Sprint(nombres) != "[Bajo eléctrico]" {
		t.Fatalf("total = %d, stems = %v", total, nombres)
	}
}

func TestStemsGlobales_Pagina(t *testing.T) {
	e := prepararStemsGlobales(t)

	total, primera := e.listar(t, dto.ListarStemsGlobalFiltro{Pagina: 1, TamanoPagina: 2})
	_, segunda := e.listar(t, dto.ListarStemsGlobalFiltro{Pagina: 2, TamanoPagina: 2})

	if total != 3 || len(primera) != 2 || len(segunda) != 1 {
		t.Fatalf("total = %d, página 1 = %v, página 2 = %v", total, primera, segunda)
	}
	for _, nombre := range segunda {
		for _, otro := range primera {
			if nombre == otro {
				t.Fatalf("%q aparece en las dos páginas", nombre)
			}
		}
	}
}
