package service

import (
	"bytes"
	"database/sql"
	"errors"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// PRU-08 (HU-SEG-B01): un integrante sin GESTIONAR_VERSIONES (rol Músico) no
// puede crear una versión, y el rechazo ocurre antes de abrir la
// transacción, así que no se inserta nada en cancionversion.

type cancionNuevaVersionFalsa struct {
	repository.CancionRepository

	iniciarLlamado bool
}

func (c *cancionNuevaVersionFalsa) ExisteCancionActivaEnProyecto(int64, int64) (bool, error) {
	return true, nil
}

func (c *cancionNuevaVersionFalsa) IniciarCreacionVersion(int64) (*sql.Tx, int, error) {
	c.iniciarLlamado = true
	return nil, 0, errors.New("no debería abrir la transacción")
}

func nuevoServicioVersiones(
	canciones *cancionNuevaVersionFalsa,
	proyectos *proyectoAccesoFalso,
) CancionService {
	return NewCancionService(
		canciones,
		proyectos,
		&integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 42}},
		nil,
	)
}

func TestCrearVersion_SinPermisoNoCreaNada(t *testing.T) {
	canciones := &cancionNuevaVersionFalsa{}
	proyectos := &proyectoAccesoFalso{sinPermiso: true}

	_, err := nuevoServicioVersiones(canciones, proyectos).CrearVersion(1, 7, 8,
		ArchivoAudio{
			Contenido:      bytes.NewReader([]byte("RIFF")),
			NombreOriginal: "x.wav",
			Tamano:         4,
		},
		nil,
	)

	if !errors.Is(err, ErrVersionSinPermiso) {
		t.Fatalf("err = %v, se esperaba ErrVersionSinPermiso", err)
	}
	if proyectos.permisoPide != "GESTIONAR_VERSIONES" {
		t.Fatalf("permiso consultado = %q", proyectos.permisoPide)
	}
	if canciones.iniciarLlamado {
		t.Fatal("no debería iniciar la creación de la versión")
	}
}

// Sin permiso, el cuerpo de la request no importa: un JSON sin archivo
// (como {"track_id": "abc123", "file_url": "x.wav"}) también recibe 403 y no
// un 400 que revele qué espera el endpoint.
func TestCrearVersion_SinPermisoYSinArchivoDevuelveSinPermiso(t *testing.T) {
	canciones := &cancionNuevaVersionFalsa{}

	_, err := nuevoServicioVersiones(canciones, &proyectoAccesoFalso{sinPermiso: true}).
		CrearVersion(1, 7, 8, ArchivoAudio{}, nil)

	if !errors.Is(err, ErrVersionSinPermiso) {
		t.Fatalf("err = %v, se esperaba ErrVersionSinPermiso", err)
	}
	if canciones.iniciarLlamado {
		t.Fatal("no debería iniciar la creación de la versión")
	}
}

func TestCrearVersion_ConPermisoSinArchivoPideLaPista(t *testing.T) {
	_, err := nuevoServicioVersiones(&cancionNuevaVersionFalsa{}, &proyectoAccesoFalso{}).
		CrearVersion(1, 7, 8, ArchivoAudio{}, nil)

	if !errors.Is(err, ErrVersionPistaObligatoria) {
		t.Fatalf("err = %v, se esperaba ErrVersionPistaObligatoria", err)
	}
}
