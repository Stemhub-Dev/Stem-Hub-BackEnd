package repository

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

func nuevoRepositorioProyectosConMock(t *testing.T) (ProyectoRepository, sqlmock.Sqlmock) {
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

	return NewProyectoRepository(db), mock
}

var columnasMisProyectos = []string{
	"codigoproyecto",
	"nombreproyecto",
	"descripcionproyecto",
	"logoproyecto",
	"nombretipoproy",
	"nombreestadoproy",
	"cantidadcanciones",
	"codrol",
	"nombrerol",
	"espropietario",
	"fechaultimamodificacion",
}

func filtroProyectosBase() dto.ListarProyectosFiltro {
	return dto.ListarProyectosFiltro{
		Busqueda:     "rock",
		Pagina:       2,
		TamanoPagina: 10,
	}
}

// Los JOIN también filtran filas (un proyecto con tipo, estado o rol
// inexistente queda afuera), así que tienen que estar en las dos consultas
// para que totalItems coincida con lo que se pagina.
func TestConsultasMisProyectosCompartenFiltro(t *testing.T) {
	condiciones := []string{
		"FROM integranteproyecto ip",
		"JOIN proyecto p",
		"JOIN tipoproyecto tp",
		"JOIN estadoproyecto ep",
		"JOIN rol r",
		"AND r.ambitorol = ip.ambitorol",
		"ip.codintegrante = $1",
		"ip.fechahorabajaintegranteproy IS NULL",
		"p.fechahorabajaproyecto IS NULL",
		"$2::text = ''",
		"p.nombreproyecto ILIKE '%' || $2::text || '%'",
		"COALESCE(array_length($3::bigint[], 1), 0) = 0",
		"p.codestadoproy = ANY($3::bigint[])",
		"COALESCE(array_length($4::bigint[], 1), 0) = 0",
		"p.codtipoproy = ANY($4::bigint[])",
	}

	for _, condicion := range condiciones {
		if !strings.Contains(consultaContarMisProyectos, condicion) {
			t.Errorf("falta %q en la consulta de conteo", condicion)
		}
		if !strings.Contains(consultaListarMisProyectos, condicion) {
			t.Errorf("falta %q en la consulta de listado", condicion)
		}
	}
}

func TestContarMisProyectos_PasaFiltros(t *testing.T) {
	repo, mock := nuevoRepositorioProyectosConMock(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\)\s+FROM integranteproyecto ip.*`+
		`p\.nombreproyecto ILIKE '%' \|\| \$2::text \|\| '%'.*`+
		`p\.codestadoproy = ANY\(\$3::bigint\[\]\).*`+
		`p\.codtipoproy = ANY\(\$4::bigint\[\]\)`).
		WithArgs(int64(42), `100\%`, []int64{2}, []int64{1, 3}).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

	filtro := filtroProyectosBase()
	filtro.Busqueda = "100%"
	filtro.Estados = []int64{2}
	filtro.Tipos = []int64{1, 3}

	total, err := repo.ContarPorIntegrante(42, filtro)

	if err != nil || total != 7 {
		t.Fatalf("total/err = %d/%v, se esperaba 7/nil", total, err)
	}
}

func TestContarMisProyectos_SinFiltrosPasaVacioYNulls(t *testing.T) {
	repo, mock := nuevoRepositorioProyectosConMock(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\)`).
		WithArgs(int64(42), "", nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	filtro := filtroProyectosBase()
	filtro.Busqueda = ""

	if _, err := repo.ContarPorIntegrante(42, filtro); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestContarMisProyectos_PropagaError(t *testing.T) {
	repo, mock := nuevoRepositorioProyectosConMock(t)
	errBase := errors.New("fallo de base")

	mock.ExpectQuery(`SELECT COUNT\(\*\)`).WillReturnError(errBase)

	if _, err := repo.ContarPorIntegrante(42, filtroProyectosBase()); !errors.Is(err, errBase) {
		t.Fatalf("err = %v, se esperaba %v", err, errBase)
	}
}

func TestListarMisProyectos_PaginaOrdenaYMapea(t *testing.T) {
	repo, mock := nuevoRepositorioProyectosConMock(t)
	fecha := time.Date(2026, 9, 20, 15, 30, 0, 0, time.UTC)
	logo := "proyectos/9/logo.png"

	mock.ExpectQuery(
		`p\.codestadoproy = ANY\(\$3::bigint\[\]\).*`+
			`p\.codtipoproy = ANY\(\$4::bigint\[\]\).*`+
			regexp.QuoteMeta(`ORDER BY actividad.fechaultimamodificacion DESC,`)+
			`\s+p\.codigoproyecto DESC\s+LIMIT \$5 OFFSET \$6`).
		WithArgs(int64(42), "rock", []int64{2}, nil, 10, 10).
		WillReturnRows(
			sqlmock.NewRows(columnasMisProyectos).
				AddRow(9, "Rock Nacional", "Primer disco", logo, "Single", "En Desarrollo", 3, 1, "Productor", true, fecha).
				AddRow(4, "Rock Alternativo", nil, nil, "Extended Play (EP)", "Sin Empezar", 0, 2, "Músico (Artista)", false, fecha),
		)

	filtro := filtroProyectosBase()
	filtro.Estados = []int64{2}

	proyectos, err := repo.ListarPorIntegrante(42, filtro, 10)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(proyectos) != 2 {
		t.Fatalf("len = %d, se esperaba 2", len(proyectos))
	}

	primero := proyectos[0]
	if primero.CodigoProyecto != 9 || primero.Nombre != "Rock Nacional" ||
		*primero.Descripcion != "Primer disco" || *primero.Logo != logo ||
		primero.Tipo != "Single" || primero.Estado != "En Desarrollo" ||
		primero.CantidadCanciones != 3 || primero.CodRol != 1 ||
		primero.NombreRol != "Productor" || !primero.EsPropietario ||
		!primero.FechaUltimaModificacion.Equal(fecha) {
		t.Fatalf("mapeo incorrecto: %+v", primero)
	}
	if primero.PortadaURL != nil {
		t.Fatal("la portada la firma el service, no el repositorio")
	}

	if proyectos[1].Descripcion != nil || proyectos[1].Logo != nil {
		t.Fatalf("los NULL deben quedar en nil: %+v", proyectos[1])
	}
}

func TestListarMisProyectos_SinFilasDevuelveSliceVacio(t *testing.T) {
	repo, mock := nuevoRepositorioProyectosConMock(t)

	mock.ExpectQuery(`LIMIT \$5 OFFSET \$6`).
		WillReturnRows(sqlmock.NewRows(columnasMisProyectos))

	proyectos, err := repo.ListarPorIntegrante(42, filtroProyectosBase(), 0)

	if err != nil || proyectos == nil || len(proyectos) != 0 {
		t.Fatalf("proyectos/err = %#v/%v, se esperaba slice vacío no nil", proyectos, err)
	}
}

func TestListarMisProyectos_PropagaError(t *testing.T) {
	repo, mock := nuevoRepositorioProyectosConMock(t)
	errBase := errors.New("fallo de base")

	mock.ExpectQuery(`LIMIT \$5 OFFSET \$6`).WillReturnError(errBase)

	if _, err := repo.ListarPorIntegrante(42, filtroProyectosBase(), 0); !errors.Is(err, errBase) {
		t.Fatalf("err = %v, se esperaba %v", err, errBase)
	}
}
