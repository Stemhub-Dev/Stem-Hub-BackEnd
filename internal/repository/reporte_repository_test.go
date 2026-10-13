package repository

import (
	"errors"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func nuevoRepositorioReportesConMock(t *testing.T) (ReporteRepository, sqlmock.Sqlmock) {
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

	return NewReporteRepository(db), mock
}

var columnasEstadoCanciones = []string{
	"codigocancion", "nombrecancion", "ultima", "versiones", "comentarios", "pendientes", "resueltos",
}

func TestObtenerEstadoCanciones_DevuelvePendientesYResueltos(t *testing.T) {
	repositorio, mock := nuevoRepositorioReportesConMock(t)
	codigoCancion := int64(4)

	mock.ExpectQuery(`FILTER \(WHERE LOWER\(ec.nombreestadocom\) = 'pendiente'\)`).
		WithArgs(int64(1), &codigoCancion, nil).
		WillReturnRows(sqlmock.NewRows(columnasEstadoCanciones).
			AddRow(int64(4), "Balada", int64(2), int64(2), int64(5), int64(3), int64(2)))

	datos, err := repositorio.ObtenerEstadoCanciones(1, ReporteFiltros{CodigoCancion: &codigoCancion})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	esperado := []map[string]interface{}{{
		"codigo_cancion":         int64(4),
		"cancion":                "Balada",
		"ultima_version":         int64(2),
		"total_versiones":        int64(2),
		"comentarios":            int64(5),
		"comentarios_pendientes": int64(3),
		"comentarios_resueltos":  int64(2),
	}}
	if !reflect.DeepEqual(datos, esperado) {
		t.Errorf("datos = %v, se esperaba %v", datos, esperado)
	}
}

func TestObtenerEstadoCanciones_PropagaErrorDeConsulta(t *testing.T) {
	repositorio, mock := nuevoRepositorioReportesConMock(t)
	falla := errors.New("falla de base")

	mock.ExpectQuery(`FROM cancion c`).WillReturnError(falla)

	if _, err := repositorio.ObtenerEstadoCanciones(1, ReporteFiltros{}); !errors.Is(err, falla) {
		t.Fatalf("se esperaba el error de la base, se obtuvo %v", err)
	}
}

func TestObtenerEstadoCanciones_PropagaErrorDeScan(t *testing.T) {
	repositorio, mock := nuevoRepositorioReportesConMock(t)

	mock.ExpectQuery(`FROM cancion c`).
		WillReturnRows(sqlmock.NewRows(columnasEstadoCanciones).
			AddRow("no es un número", "Balada", 1, 1, 1, 1, 1))

	if _, err := repositorio.ObtenerEstadoCanciones(1, ReporteFiltros{}); err == nil {
		t.Fatal("se esperaba un error de scan")
	}
}
