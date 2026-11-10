package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

var columnasPreguntaFrecuente = []string{
	"codigopreguntafrecuente",
	"funcionalidadprincipal",
	"preguntafrecuente",
	"respuestafrecuente",
}

func nuevoRepositorioPreguntasConMock(t *testing.T) (PreguntaFrecuenteRepository, sqlmock.Sqlmock) {
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

	return NewPreguntaFrecuenteRepository(db), mock
}

func TestListarPreguntasFrecuentes_SoloActivasEnOrdenDeCarga(t *testing.T) {
	repo, mock := nuevoRepositorioPreguntasConMock(t)

	mock.ExpectQuery(`WHERE fechahorabajapreguntafrecuente IS NULL\s+ORDER BY codigopreguntafrecuente`).
		WillReturnRows(sqlmock.NewRows(columnasPreguntaFrecuente).
			AddRow(1, "Registro y acceso", "¿Cómo creo una cuenta?", "Con tu email."))

	preguntas, err := repo.ListarActivas()

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(preguntas) != 1 || preguntas[0].RespuestaFrecuente != "Con tu email." {
		t.Fatalf("preguntas = %+v", preguntas)
	}
}

func TestListarPreguntasFrecuentes_SinFilasDevuelveSliceVacio(t *testing.T) {
	repo, mock := nuevoRepositorioPreguntasConMock(t)

	mock.ExpectQuery(`FROM preguntafrecuente`).
		WillReturnRows(sqlmock.NewRows(columnasPreguntaFrecuente))

	preguntas, err := repo.ListarActivas()

	if err != nil || preguntas == nil || len(preguntas) != 0 {
		t.Fatalf("preguntas/err = %#v/%v, se esperaba slice vacío no nil", preguntas, err)
	}
}

func TestListarPreguntasFrecuentes_PropagaError(t *testing.T) {
	repo, mock := nuevoRepositorioPreguntasConMock(t)
	errBase := errors.New("fallo de base")

	mock.ExpectQuery(`FROM preguntafrecuente`).WillReturnError(errBase)

	if _, err := repo.ListarActivas(); !errors.Is(err, errBase) {
		t.Fatalf("err = %v, se esperaba %v", err, errBase)
	}
}
