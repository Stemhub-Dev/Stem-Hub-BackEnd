package repository

import (
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

// convertidorDirecto deja pasar los argumentos tal cual, igual que hace el
// driver pgx con el bigint[] del filtro por proyecto; el convertidor por
// defecto de database/sql rechazaría un []int64.
type convertidorDirecto struct{}

func (convertidorDirecto) ConvertValue(valor any) (driver.Value, error) {
	return valor, nil
}

func nuevoRepositorioConMock(t *testing.T) (CancionRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(
		sqlmock.ValueConverterOption(convertidorDirecto{}),
	)
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

// filtroBase arma el filtro con todos los criterios puestos; cada test ajusta
// lo que necesita.
func filtroBase() dto.ListarMisCancionesFiltro {
	return dto.ListarMisCancionesFiltro{
		Busqueda:     "bal",
		Orden:        dto.OrdenMisCancionesReciente,
		Pagina:       2,
		TamanoPagina: 10,
	}
}

// El conteo y el listado repiten el FROM/WHERE porque ambas consultas se
// escriben completas (sin concatenar fragmentos). Si una se edita y la otra
// no, totalItems dejaría de reflejar lo que se pagina.
func TestConsultasMisCancionesCompartenFiltro(t *testing.T) {
	condiciones := []string{
		"FROM integranteproyecto ip",
		"INNER JOIN proyecto p",
		"INNER JOIN cancion c",
		"ip.codintegrante = $1",
		"ip.fechahorabajaintegranteproy IS NULL",
		"p.fechahorabajaproyecto IS NULL",
		"c.fechahorabajacancion IS NULL",
		"$2::text = ''",
		"c.nombrecancion ILIKE '%' || $2::text || '%'",
		"COALESCE(array_length($3::bigint[], 1), 0) = 0",
		"c.codigoproyecto = ANY($3::bigint[])",
	}

	for _, condicion := range condiciones {
		if !strings.Contains(consultaContarMisCanciones, condicion) {
			t.Errorf("falta %q en la consulta de conteo", condicion)
		}
		if !strings.Contains(consultaListarMisCanciones, condicion) {
			t.Errorf("falta %q en la consulta de listado", condicion)
		}
	}
}

// El ORDER BY se resuelve con CASE sobre un parámetro: los valores de orden
// nunca se concatenan a la consulta.
func TestConsultaListarCubreTodosLosOrdenes(t *testing.T) {
	for _, orden := range []string{
		dto.OrdenMisCancionesReciente,
		dto.OrdenMisCancionesNombreAsc,
		dto.OrdenMisCancionesNombreDesc,
	} {
		if !strings.Contains(consultaListarMisCanciones, "$4::text = '"+orden+"'") {
			t.Errorf("la consulta no contempla el orden %q", orden)
		}
	}
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

func TestProyectosComoArreglo(t *testing.T) {
	if proyectosComoArreglo(nil) != nil {
		t.Error("un filtro sin proyectos debe viajar como NULL")
	}
	if proyectosComoArreglo([]int64{}) != nil {
		t.Error("un filtro con lista vacía debe viajar como NULL")
	}

	arreglo, ok := proyectosComoArreglo([]int64{3, 7}).([]int64)
	if !ok || len(arreglo) != 2 || arreglo[0] != 3 || arreglo[1] != 7 {
		t.Errorf("proyectosComoArreglo = %v, se esperaba []int64{3, 7}", arreglo)
	}
}

func TestContarPorIntegrante_AplicaFiltroYBusquedaEscapada(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\)\s+FROM integranteproyecto ip.*` +
		`ip\.codintegrante = \$1.*` +
		`ip\.fechahorabajaintegranteproy IS NULL.*` +
		`p\.fechahorabajaproyecto IS NULL.*` +
		`c\.fechahorabajacancion IS NULL.*` +
		`c\.nombrecancion ILIKE '%' \|\| \$2::text \|\| '%'.*` +
		`c\.codigoproyecto = ANY\(\$3::bigint\[\]\)`).
		WithArgs(int64(42), `50\%`, []int64{9}).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	filtro := filtroBase()
	filtro.Busqueda = "50%"
	filtro.Proyectos = []int64{9}

	total, err := repo.ContarPorIntegrante(42, filtro)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, se esperaba 3", total)
	}
}

func TestContarPorIntegrante_SinFiltrosPasaVaciosYNull(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\)`).
		WithArgs(int64(42), "", nil).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	filtro := filtroBase()
	filtro.Busqueda = ""
	filtro.Proyectos = nil

	total, err := repo.ContarPorIntegrante(42, filtro)

	if err != nil || total != 0 {
		t.Fatalf("total/err = %d/%v, se esperaba 0/nil", total, err)
	}
}

func TestContarPorIntegrante_PropagaError(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)
	errBase := errors.New("fallo de base")

	mock.ExpectQuery(`SELECT COUNT\(\*\)`).WillReturnError(errBase)

	if _, err := repo.ContarPorIntegrante(42, filtroBase()); !errors.Is(err, errBase) {
		t.Fatalf("err = %v, se esperaba %v", err, errBase)
	}
}

func TestListarPorIntegrante_PasaFiltrosOrdenYPaginacion(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)

	mock.ExpectQuery(
		`ip\.codintegrante = \$1.*` +
			`c\.nombrecancion ILIKE '%' \|\| \$2::text \|\| '%'.*` +
			`c\.codigoproyecto = ANY\(\$3::bigint\[\]\).*` +
			regexp.QuoteMeta(`CASE WHEN $4::text = 'nombreAsc' THEN lower(c.nombrecancion) END ASC`) +
			`.*` +
			regexp.QuoteMeta(`c.codigocancion DESC`) +
			`\s+LIMIT \$5 OFFSET \$6`).
		WithArgs(int64(42), "bal", []int64{9, 4}, dto.OrdenMisCancionesNombreAsc, 10, 10).
		WillReturnRows(
			sqlmock.NewRows(columnasMisCanciones).
				AddRow(5, "Balada", 9, "Disco", 50, 3, "proyectos/9/canciones/5/v3.mp3", "mp3").
				AddRow(4, "Balada sin versión", 9, "Disco", nil, nil, nil, nil),
		)

	filtro := filtroBase()
	filtro.Proyectos = []int64{9, 4}
	filtro.Orden = dto.OrdenMisCancionesNombreAsc

	canciones, err := repo.ListarPorIntegrante(42, filtro, 10)

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

	mock.ExpectQuery(`LIMIT \$5 OFFSET \$6`).
		WithArgs(int64(42), "", nil, dto.OrdenMisCancionesReciente, 10, 0).
		WillReturnRows(sqlmock.NewRows(columnasMisCanciones))

	filtro := filtroBase()
	filtro.Busqueda = ""
	filtro.Pagina = 1

	canciones, err := repo.ListarPorIntegrante(42, filtro, 0)

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

	mock.ExpectQuery(`LIMIT \$5 OFFSET \$6`).WillReturnError(errBase)

	if _, err := repo.ListarPorIntegrante(42, filtroBase(), 0); !errors.Is(err, errBase) {
		t.Fatalf("err = %v, se esperaba %v", err, errBase)
	}
}
