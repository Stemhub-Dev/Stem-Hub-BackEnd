package repository

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestEliminarStem_BorraRespuestasYComentariosYDaDeBajaElStem(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("no se pudo crear sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	esperarContextoStem(mock, `{"nombre":"Bajo"}`)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM comentariorespuesta")).
		WithArgs(int64(10)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM comentario WHERE codstem = $1")).
		WithArgs(int64(10)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SET fechahorabajastem = CURRENT_TIMESTAMP")).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"fechahorabajastem"}).AddRow(time.Now()))
	esperarAuditoriaStem(mock)
	mock.ExpectCommit()

	if err := NewStemRepository(db).Eliminar(1, 7, 10); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas no cumplidas: %v", err)
	}
}

func TestActualizarStem_CreaLaCategoriaNuevaYLaAsignaEnLaMismaTransaccion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("no se pudo crear sqlmock: %v", err)
	}
	defer db.Close()

	nueva := "Sintes"

	mock.ExpectBegin()
	esperarContextoStem(mock, `{"nombre":"Viejo"}`)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO categoriastem")).
		WithArgs("Sintes", int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"codcategoriastem"}).AddRow(int64(31)))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE stem")).
		WithArgs(int64(10), "Bajo", "bajo.wav", int64(31)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	esperarContextoStem(mock, `{"nombre":"Bajo"}`)
	esperarAuditoriaStem(mock)
	mock.ExpectCommit()

	err = NewStemRepository(db).Actualizar(1, 7, 10, CambiosStem{
		Nombre:           "Bajo",
		NombreArchivo:    "bajo.wav",
		CodCategoriaStem: 2,
		NuevaCategoria:   &nueva,
	})

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas no cumplidas: %v", err)
	}
}

// esperarContextoStem cubre la foto del stem que se toma para la auditoría.
func esperarContextoStem(mock sqlmock.Sqlmock, datos string) {
	mock.ExpectQuery("jsonb_build_object").
		WithArgs(int64(10), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"codigoproyecto", "nombreproyecto", "nombrestem", "datos"}).
			AddRow(int64(7), "Proyecto", "Bajo", datos))
}

func esperarAuditoriaStem(mock sqlmock.Sqlmock) {
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO public.auditoria")).
		WillReturnResult(sqlmock.NewResult(0, 1))
}
