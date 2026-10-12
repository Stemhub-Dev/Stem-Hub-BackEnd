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
// puede crear una versión, y el rechazo ocurre antes de validar el archivo y
// de abrir la transacción, así que no se inserta nada en cancionversion.

var errRepositorio = errors.New("falla del repositorio")

type cancionNuevaVersionFalsa struct {
	repository.CancionRepository

	noExiste       bool
	errExiste      error
	iniciarLlamado bool
}

func (c *cancionNuevaVersionFalsa) ExisteCancionActivaEnProyecto(int64, int64) (bool, error) {
	return !c.noExiste, c.errExiste
}

func (c *cancionNuevaVersionFalsa) IniciarCreacionVersion(int64) (*sql.Tx, int, error) {
	c.iniciarLlamado = true
	return nil, 0, errRepositorio
}

type permisoVersionFalso struct {
	repository.ProyectoRepository

	sinPermiso  bool
	err         error
	permisoPide string
}

func (p *permisoVersionFalso) PuedeRealizarEnProyecto(_, _ int64, permiso string) (bool, error) {
	p.permisoPide = permiso
	return !p.sinPermiso, p.err
}

type escenarioVersion struct {
	canciones   *cancionNuevaVersionFalsa
	proyectos   *permisoVersionFalso
	integrantes *integranteRepositoryFalso
}

func nuevoEscenarioVersion() escenarioVersion {
	return escenarioVersion{
		canciones:   &cancionNuevaVersionFalsa{},
		proyectos:   &permisoVersionFalso{},
		integrantes: &integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 42}},
	}
}

func (e escenarioVersion) crear(archivo ArchivoAudio, notas *string) error {
	_, err := NewCancionService(e.canciones, e.proyectos, e.integrantes, nil).
		CrearVersion(1, 7, 8, archivo, notas)
	return err
}

func wavValido() ArchivoAudio {
	return ArchivoAudio{
		Contenido:      bytes.NewReader([]byte("RIFF")),
		NombreOriginal: "x.wav",
		Tamano:         4,
	}
}

func TestCrearVersion_SinPermisoNoCreaNada(t *testing.T) {
	e := nuevoEscenarioVersion()
	e.proyectos.sinPermiso = true

	err := e.crear(wavValido(), nil)

	if !errors.Is(err, ErrVersionSinPermiso) {
		t.Fatalf("err = %v, se esperaba ErrVersionSinPermiso", err)
	}
	if e.proyectos.permisoPide != "GESTIONAR_VERSIONES" {
		t.Fatalf("permiso consultado = %q", e.proyectos.permisoPide)
	}
	if e.canciones.iniciarLlamado {
		t.Fatal("no debería iniciar la creación de la versión")
	}
}

// Sin permiso, el cuerpo de la request no importa: un JSON sin archivo
// (como {"track_id": "abc123", "file_url": "x.wav"}) también recibe 403 y no
// un 400 que revele qué espera el endpoint.
func TestCrearVersion_SinPermisoYSinArchivoDevuelveSinPermiso(t *testing.T) {
	e := nuevoEscenarioVersion()
	e.proyectos.sinPermiso = true

	err := e.crear(ArchivoAudio{}, nil)

	if !errors.Is(err, ErrVersionSinPermiso) {
		t.Fatalf("err = %v, se esperaba ErrVersionSinPermiso", err)
	}
	if e.canciones.iniciarLlamado {
		t.Fatal("no debería iniciar la creación de la versión")
	}
}

// Las verificaciones de acceso cortan antes de mirar el archivo, y con
// acceso el archivo se valida antes de abrir la transacción.
func TestCrearVersion_ValidacionesEnOrden(t *testing.T) {
	notasEnBlanco := "   "
	notasConTexto := "  Mezcla nueva  "

	casos := []struct {
		nombre   string
		preparar func(*escenarioVersion)
		archivo  ArchivoAudio
		notas    *string
		esperado error
		abreTx   bool
	}{
		{
			nombre:   "canción inexistente",
			preparar: func(e *escenarioVersion) { e.canciones.noExiste = true },
			esperado: ErrVersionCancionNoEncontrada,
		},
		{
			nombre:   "falla al buscar la canción",
			preparar: func(e *escenarioVersion) { e.canciones.errExiste = errRepositorio },
			esperado: errRepositorio,
		},
		{
			nombre:   "usuario sin perfil",
			preparar: func(e *escenarioVersion) { e.integrantes.err = sql.ErrNoRows },
			esperado: ErrCancionPerfilRequerido,
		},
		{
			nombre:   "falla al buscar el integrante",
			preparar: func(e *escenarioVersion) { e.integrantes.err = errRepositorio },
			esperado: errRepositorio,
		},
		{
			nombre:   "falla al consultar el permiso",
			preparar: func(e *escenarioVersion) { e.proyectos.err = errRepositorio },
			esperado: errRepositorio,
		},
		{
			nombre:   "con permiso y sin archivo",
			esperado: ErrVersionPistaObligatoria,
		},
		{
			nombre: "archivo demasiado grande",
			archivo: ArchivoAudio{
				Contenido:      bytes.NewReader(nil),
				NombreOriginal: "x.wav",
				Tamano:         TamanoMaximoArchivoAudio + 1,
			},
			esperado: ErrCancionArchivoDemasiadoGrande,
		},
		{
			nombre: "formato no soportado",
			archivo: ArchivoAudio{
				Contenido:      bytes.NewReader(nil),
				NombreOriginal: "x.ogg",
				Tamano:         4,
			},
			esperado: ErrCancionFormatoInvalido,
		},
		{
			nombre:   "notas con texto llegan a abrir la transacción",
			archivo:  wavValido(),
			notas:    &notasConTexto,
			esperado: errRepositorio,
			abreTx:   true,
		},
		{
			nombre:   "notas en blanco llegan a abrir la transacción",
			archivo:  wavValido(),
			notas:    &notasEnBlanco,
			esperado: errRepositorio,
			abreTx:   true,
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioVersion()
			if caso.preparar != nil {
				caso.preparar(&e)
			}

			err := e.crear(caso.archivo, caso.notas)

			if !errors.Is(err, caso.esperado) {
				t.Fatalf("err = %v, se esperaba %v", err, caso.esperado)
			}
			if e.canciones.iniciarLlamado != caso.abreTx {
				t.Fatalf("abrió la transacción = %v, se esperaba %v", e.canciones.iniciarLlamado, caso.abreTx)
			}
		})
	}
}
