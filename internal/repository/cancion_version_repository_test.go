package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// HU-VER-F08: el encabezado de la pista muestra el autor de la versión y
// marca la actual.

var columnasVersiones = []string{
	"codigocancionversion",
	"numeroversion",
	"fechahoraaltaversion",
	"urlarchivocancionver",
	"formatoarchivocancionver",
	"notasversion",
	"codintegrante",
	"nombreintegrante",
	"esactual",
}

func TestListarVersiones_DevuelveAutorYVersionActual(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)
	alta := time.Date(2026, 10, 5, 14, 30, 0, 0, time.UTC)

	mock.ExpectQuery(`LEFT JOIN integrante i\s+ON i.codintegrante = cv.codintegrantealtaversion`).
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows(columnasVersiones).
			AddRow(int64(21), 2, alta, "k2.wav", "wav", "Mezcla nueva", int64(42), "Ana", true).
			AddRow(int64(20), 1, alta, "k1.wav", "wav", nil, nil, nil, false))

	versiones, err := repo.ListarVersiones(8)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(versiones) != 2 {
		t.Fatalf("se esperaban 2 versiones, llegaron %d", len(versiones))
	}

	actual := versiones[0]
	if !actual.EsActual || actual.EtiquetaVersion != "v1.1.0" {
		t.Errorf("la versión 2 debería ser la actual v1.1.0: %+v", actual)
	}
	if actual.Autor == nil || actual.Autor.CodigoIntegrante != 42 || actual.Autor.Nombre != "Ana" {
		t.Errorf("autor inesperado: %+v", actual.Autor)
	}

	anterior := versiones[1]
	if anterior.EsActual {
		t.Error("la versión 1 no debería ser la actual")
	}
	// Versión subida antes de guardar el autor.
	if anterior.Autor != nil {
		t.Errorf("la versión sin autor debería devolver autor nil: %+v", anterior.Autor)
	}
}

func TestInsertarVersion_GuardaElAutor(t *testing.T) {
	repo, mock := nuevoRepositorioConMock(t)
	notas := "Mezcla nueva"

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO cancionversion \(\s+codigocancion,\s+numeroversion,\s+fechahoraaltaversion,\s+codintegrantealtaversion,`).
		WithArgs(int64(8), 3, int64(42), "k3.wav", "wav", &notas).
		WillReturnRows(sqlmock.NewRows([]string{"codigocancionversion"}).AddRow(int64(30)))
	mock.ExpectRollback()

	impl, _ := repo.(*cancionRepository)
	tx, err := impl.db.Begin()
	if err != nil {
		t.Fatalf("no se pudo abrir la transacción: %v", err)
	}
	defer tx.Rollback()

	codigo, err := repo.InsertarVersion(tx, 8, 3, 42, "k3.wav", "wav", &notas)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if codigo != 30 {
		t.Errorf("codigo = %d, se esperaba 30", codigo)
	}
}
