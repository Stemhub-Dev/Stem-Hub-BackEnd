package repository

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func nuevoComentarioRepositoryMock(t *testing.T) (ComentarioRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("no se pudo crear sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return NewComentarioRepository(db), mock
}

func TestListarComentarios_LeeElAvatarDelAutorYDeLasRespuestas(t *testing.T) {
	repo, mock := nuevoComentarioRepositoryMock(t)
	ahora := time.Now()

	columnas := []string{
		"codigocomentario", "codstem", "descripcioncomentario", "nombreestadocom",
		"tiempoiniciosegundos", "tiempofinsegundos", "fechahoraaltacomentario",
		"codintegrante", "nombreintegrante", "urlavatarintegrante", "espropio",
		"codigorespuestacomentario", "descripcionrespuesta", "fechahoraaltarespuesta",
		"codintegrante", "nombreintegrante", "urlavatarintegrante", "espropiarespuesta",
	}
	mock.ExpectQuery(regexp.QuoteMeta("i.urlavatarintegrante")).
		WithArgs(int64(9), int64(42)).
		WillReturnRows(sqlmock.NewRows(columnas).
			// Comentario con avatar y dos respuestas: una con avatar, otra sin.
			AddRow(1, nil, "Subí la voz", "Pendiente", nil, nil, ahora,
				7, "Juan", "avatares/7.png", false,
				70, "Dale", ahora, 42, "Ana", "avatares/42.png", true).
			AddRow(1, nil, "Subí la voz", "Pendiente", nil, nil, ahora,
				7, "Juan", "avatares/7.png", false,
				71, "Listo", ahora, 8, "Leo", nil, false).
			// Comentario sin avatar ni respuestas.
			AddRow(2, 30, "Bajo", "Resuelto", 1.5, 2.5, ahora,
				8, "Leo", nil, false,
				nil, nil, nil, nil, nil, nil, nil))

	comentarios, err := repo.ListarPorVersion(9, 42)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if len(comentarios) != 2 || len(comentarios[0].Respuestas) != 2 || len(comentarios[1].Respuestas) != 0 {
		t.Fatalf("comentarios mal agrupados: %+v", comentarios)
	}
	if k := comentarios[0].Autor.AvatarObjectKey; k == nil || *k != "avatares/7.png" {
		t.Fatalf("avatar del autor = %v", k)
	}
	if k := comentarios[0].Respuestas[0].Autor.AvatarObjectKey; k == nil || *k != "avatares/42.png" {
		t.Fatalf("avatar de la respuesta = %v", k)
	}
	if comentarios[0].Respuestas[1].Autor.AvatarObjectKey != nil || comentarios[1].Autor.AvatarObjectKey != nil {
		t.Fatal("sin avatar en la base debería quedar nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas no cumplidas: %v", err)
	}
}

func TestObtenerAutorRespuesta_ExigeRespuestaYComentarioActivosEnLaVersion(t *testing.T) {
	repo, mock := nuevoComentarioRepositoryMock(t)

	mock.ExpectQuery(regexp.QuoteMeta("FROM comentariorespuesta cr")).
		WithArgs(int64(20), int64(10), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"codintegrante"}).AddRow(42))

	autor, err := repo.ObtenerAutorRespuesta(20, 10, 9)
	if err != nil || autor != 42 {
		t.Fatalf("autor = %d, err = %v", autor, err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("cr.fechahorabajarespuesta IS NULL")).
		WithArgs(int64(21), int64(10), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"codintegrante"}))

	if _, err := repo.ObtenerAutorRespuesta(21, 10, 9); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err = %v, se esperaba sql.ErrNoRows", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas no cumplidas: %v", err)
	}
}

func TestDarDeBajaRespuesta_EsBajaLogica(t *testing.T) {
	repo, mock := nuevoComentarioRepositoryMock(t)
	baja := regexp.QuoteMeta("UPDATE comentariorespuesta SET fechahorabajarespuesta = CURRENT_TIMESTAMP")

	mock.ExpectExec(baja).WithArgs(int64(20)).WillReturnResult(sqlmock.NewResult(0, 1))
	if eliminada, err := repo.DarDeBajaRespuesta(20); err != nil || !eliminada {
		t.Fatalf("eliminada = %v, err = %v", eliminada, err)
	}

	// Ya dada de baja: no afecta filas.
	mock.ExpectExec(baja).WithArgs(int64(20)).WillReturnResult(sqlmock.NewResult(0, 0))
	if eliminada, err := repo.DarDeBajaRespuesta(20); err != nil || eliminada {
		t.Fatalf("eliminada = %v, err = %v", eliminada, err)
	}

	mock.ExpectExec(baja).WithArgs(int64(20)).WillReturnError(errors.New("db caída"))
	if _, err := repo.DarDeBajaRespuesta(20); err == nil {
		t.Fatal("se esperaba el error de la base")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas no cumplidas: %v", err)
	}
}
