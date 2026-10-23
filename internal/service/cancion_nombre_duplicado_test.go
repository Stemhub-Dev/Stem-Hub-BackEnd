package service

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// HU-ABM-02-01 / HU-ABM-02-02: si otra request con el mismo nombre ganó la
// carrera, el repositorio lo detecta dentro de la transacción y el service
// responde lo mismo que con el chequeo previo (409), no un 500.

type cancionNombreCarreraFalsa struct {
	repository.CancionRepository
}

func (cancionNombreCarreraFalsa) ExisteNombreEnProyecto(int64, string) (bool, error) {
	return false, nil
}

func (cancionNombreCarreraFalsa) IniciarCreacionCancion(int64, string) (*sql.Tx, int64, error) {
	return nil, 0, repository.ErrNombreCancionDuplicado
}

func (cancionNombreCarreraFalsa) ExisteCancionActivaEnProyecto(int64, int64) (bool, error) {
	return true, nil
}

func (cancionNombreCarreraFalsa) ExisteNombreEnProyectoExceptoCancion(int64, int64, string) (bool, error) {
	return false, nil
}

func (cancionNombreCarreraFalsa) ActualizarNombre(int64, int64, string) error {
	return repository.ErrNombreCancionDuplicado
}

type proyectoGestionableFalso struct {
	repository.ProyectoRepository
}

func (proyectoGestionableFalso) ExisteProyectoActivo(int64) (bool, error) {
	return true, nil
}

func (proyectoGestionableFalso) PuedeGestionarCanciones(int64, int64) (bool, error) {
	return true, nil
}

func servicioNombreCarrera() CancionService {
	return NewCancionService(
		cancionNombreCarreraFalsa{},
		proyectoGestionableFalso{},
		&integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 42}},
		nil,
	)
}

func TestCrear_NombreTomadoEnParaleloDevuelveDuplicado(t *testing.T) {
	_, err := servicioNombreCarrera().Crear(1, 7, "Balada", wavValido())

	if !errors.Is(err, ErrCancionNombreDuplicado) {
		t.Fatalf("err = %v, quería ErrCancionNombreDuplicado", err)
	}
}

func TestEditar_NombreTomadoEnParaleloDevuelveDuplicado(t *testing.T) {
	_, err := servicioNombreCarrera().Editar(1, 7, 8, dto.EditarCancionRequest{Nombre: "Balada"})

	if !errors.Is(err, ErrCancionNombreDuplicado) {
		t.Fatalf("err = %v, quería ErrCancionNombreDuplicado", err)
	}
}
