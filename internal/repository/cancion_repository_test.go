package repository

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func nuevoRepositorioConMock(t *testing.T) (CancionRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("no se pudo crear sqlmock: %v", err)
	}

	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectativas no cumplidas: %v", err)
		}
		db.Close()
	})

	return NewCancionRepository(db), mock
}

var columnasMisCanciones = []string{
	"codigocancion",
	"nombrecancion",
	"codigoproyecto",
	"nombreproyecto",
	"codigocancionversion",
	"numeroversion",
	"urlarchivocancionver",
	"formatoarchivocancionver",
}

func TestEscaparPatronLike(t *testing.T) {
	casos := map[string]string{
		"":          "",
		"balada":    "balada",
		"100%":      `100\%`,
		"mi_tema":   `mi\_tema`,
		`a\b`:       `a\\b`,
		`%_\`:       `\%\_\\`,
		"Canción 1": "Canción 1",
	}

	for entrada, esperado := range casos {
		if obtenido := escaparPatronLike(entrada); obtenido != esperado {
			t.Errorf("escaparPatronLike(%q) = %q, se esperaba %q", entrada, obtenido, esperado)
		}
	}
}

func TestContarPorIntegrante_AplicaFiltroYBusquedaEscapada(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\)\s+FROM integranteproyecto ip.*` +
		`ip\.codintegrante = \$1.*` +
		`ip\.fechahorabajaintegranteproy IS NULL.*` +
		`p\.fechahorabajaproyecto IS NULL.*` +
		`c\.fechahorabajacancion IS NULL.*` +
		`c\.nombrecancion ILIKE '%' \|\| \$2::text \|\| '%'`).
		WithArgs(int64(42), `50\%`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	total, err := repo.ContarPorIntegrante(42, "50%")

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, se esperaba 3", total)
	}
}

func TestContarPorIntegrante_SinBusquedaPasaCadenaVacia(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\)`).
		WithArgs(int64(42), "").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	total, err := repo.ContarPorIntegrante(42, "")

	if err != nil || total != 0 {
		t.Fatalf("total/err = %d/%v, se esperaba 0/nil", total, err)
	}
}

func TestContarPorIntegrante_PropagaError(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)
	errBase := errors.New("fallo de base")

	mock.ExpectQuery(`SELECT COUNT\(\*\)`).WillReturnError(errBase)

	if _, err := repo.ContarPorIntegrante(42, ""); !errors.Is(err, errBase) {
		t.Fatalf("err = %v, se esperaba %v", err, errBase)
	}
}

func TestListarPorIntegrante_PaginaOrdenaYMapeaFilas(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)

	mock.ExpectQuery(
		`ip\.codintegrante = \$1.*` +
			`c\.nombrecancion ILIKE '%' \|\| \$2::text \|\| '%'.*` +
			regexp.QuoteMeta(`ORDER BY cv.fechahoraaltaversion DESC NULLS LAST,`) +
			`\s+c\.codigocancion DESC\s+LIMIT \$3 OFFSET \$4`).
		WithArgs(int64(42), "bal", 10, 10).
		WillReturnRows(
			sqlmock.NewRows(columnasMisCanciones).
				AddRow(5, "Balada", 9, "Disco", 50, 3, "proyectos/9/canciones/5/v3.mp3", "mp3").
				AddRow(4, "Balada sin versión", 9, "Disco", nil, nil, nil, nil),
		)

	canciones, err := repo.ListarPorIntegrante(42, "bal", 10, 10)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(canciones) != 2 {
		t.Fatalf("len = %d, se esperaba 2", len(canciones))
	}

	conVersion := canciones[0]
	if conVersion.CodigoCancion != 5 || conVersion.Nombre != "Balada" ||
		conVersion.CodigoProyecto != 9 || conVersion.NombreProyecto != "Disco" {
		t.Fatalf("mapeo incorrecto: %+v", conVersion)
	}
	if conVersion.VersionActual == nil ||
		conVersion.VersionActual.NumeroVersion != 3 ||
		conVersion.VersionActual.EtiquetaVersion != "v1.2.0" ||
		*conVersion.VersionActual.FormatoArchivo != "mp3" {
		t.Fatalf("versión actual incorrecta: %+v", conVersion.VersionActual)
	}

	if canciones[1].VersionActual != nil {
		t.Fatal("una canción sin versiones debe tener versionActual nil")
	}
}

func TestListarPorIntegrante_SinFilasDevuelveSliceVacio(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)

	mock.ExpectQuery(`LIMIT \$3 OFFSET \$4`).
		WithArgs(int64(42), "", 10, 0).
		WillReturnRows(sqlmock.NewRows(columnasMisCanciones))

	canciones, err := repo.ListarPorIntegrante(42, "", 10, 0)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if canciones == nil || len(canciones) != 0 {
		t.Fatalf("canciones = %#v, se esperaba slice vacío no nil", canciones)
	}
}

func TestListarPorIntegrante_PropagaError(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)
	errBase := errors.New("fallo de base")

	mock.ExpectQuery(`LIMIT \$3 OFFSET \$4`).WillReturnError(errBase)

	if _, err := repo.ListarPorIntegrante(42, "", 10, 0); !errors.Is(err, errBase) {
		t.Fatalf("err = %v, se esperaba %v", err, errBase)
	}
}
