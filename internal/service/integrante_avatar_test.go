package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

type perfilRepositoryFalso struct {
	repository.IntegranteRepository

	integrante model.Integrante

	avatarGuardado      *string
	nombreGuardado      string
	descripcionGuardada *string
	actualizado         bool
}

func (r *perfilRepositoryFalso) BuscarPorCodigoUsuario(int64) (*model.Integrante, error) {
	copia := r.integrante
	return &copia, nil
}

func (r *perfilRepositoryFalso) ActualizarPerfil(_ int64, nombre string, descripcion *string, avatar *string) error {
	r.actualizado = true
	r.nombreGuardado = nombre
	r.descripcionGuardada = descripcion
	r.avatarGuardado = avatar
	return nil
}

func nuevoServicioIntegrante(avatar *string) (*perfilRepositoryFalso, *storageStemFalso, IntegranteService) {
	repo := &perfilRepositoryFalso{
		integrante: model.Integrante{
			CodIntegrante:    7,
			NombreIntegrante: "Ana",
			AvatarObjectKey:  avatar,
		},
	}
	storage := &storageStemFalso{}
	return repo, storage, NewIntegranteService(repo, storage)
}

func TestEditarPerfil_QuitarAvatarLimpiaFilaYBorraArchivo(t *testing.T) {
	repo, storage, svc := nuevoServicioIntegrante(texto("integrantes/7/avatar.jpg"))

	integrante, avatarUrl, err := svc.EditarPerfil(1, "Ana", nil, nil, true)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.avatarGuardado != nil || integrante.AvatarObjectKey != nil || avatarUrl != nil {
		t.Fatalf("esperaba avatar en NULL, guardado=%v", repo.avatarGuardado)
	}
	if len(storage.eliminados) != 1 || storage.eliminados[0] != "integrantes/7/avatar.jpg" {
		t.Fatalf("archivos eliminados = %v", storage.eliminados)
	}
}

func TestEditarPerfil_SinPedirQuitarConservaAvatar(t *testing.T) {
	repo, storage, svc := nuevoServicioIntegrante(texto("integrantes/7/avatar.jpg"))

	if _, _, err := svc.EditarPerfil(1, "Ana", nil, nil, false); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.avatarGuardado == nil || *repo.avatarGuardado != "integrantes/7/avatar.jpg" {
		t.Fatalf("debería conservar el avatar, guardado=%v", repo.avatarGuardado)
	}
	if len(storage.eliminados) != 0 {
		t.Fatalf("no debería borrar archivos: %v", storage.eliminados)
	}
}

func TestEditarPerfil_QuitarSinAvatarNoBorraNada(t *testing.T) {
	_, storage, svc := nuevoServicioIntegrante(nil)

	if _, _, err := svc.EditarPerfil(1, "Ana", nil, nil, true); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(storage.eliminados) != 0 {
		t.Fatalf("no debería borrar archivos: %v", storage.eliminados)
	}
}

func TestEditarPerfil_AvatarNuevoTienePrioridadSobreQuitar(t *testing.T) {
	repo, _, svc := nuevoServicioIntegrante(nil)

	_, _, err := svc.EditarPerfil(1, "Ana", nil, &ArchivoImagen{
		Contenido:      strings.NewReader("img"),
		NombreOriginal: "foto.png",
		Tamano:         3,
	}, true)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.avatarGuardado == nil || *repo.avatarGuardado != "integrantes/7/avatar.png" {
		t.Fatalf("esperaba el avatar nuevo, guardado=%v", repo.avatarGuardado)
	}
}

func TestEditarPerfil_PerfilDadoDeBaja(t *testing.T) {
	repo, _, svc := nuevoServicioIntegrante(texto("integrantes/7/avatar.jpg"))
	baja := time.Now()
	repo.integrante.FechaHoraBajaIntegrante = &baja

	if _, _, err := svc.EditarPerfil(1, "Ana", nil, nil, true); err != ErrPerfilNoEncontrado {
		t.Fatalf("esperaba ErrPerfilNoEncontrado, obtuve %v", err)
	}
}

func TestEditarPerfil_CambioDeFormatoBorraAvatarAnterior(t *testing.T) {
	_, storage, svc := nuevoServicioIntegrante(texto("integrantes/7/avatar.png"))

	_, _, err := svc.EditarPerfil(1, "Ana", nil, &ArchivoImagen{
		Contenido:      strings.NewReader("img"),
		NombreOriginal: "foto.jpg",
		Tamano:         3,
	}, false)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(storage.eliminados) != 1 || storage.eliminados[0] != "integrantes/7/avatar.png" {
		t.Fatalf("archivos eliminados = %v", storage.eliminados)
	}
}

func TestEditarPerfil_MismoFormatoNoBorraElArchivoSobrescrito(t *testing.T) {
	_, storage, svc := nuevoServicioIntegrante(texto("integrantes/7/avatar.jpg"))

	_, _, err := svc.EditarPerfil(1, "Ana", nil, &ArchivoImagen{
		Contenido:      strings.NewReader("img"),
		NombreOriginal: "foto.JPG",
		Tamano:         3,
	}, false)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(storage.eliminados) != 0 {
		t.Fatalf("no debería borrar el avatar recién subido: %v", storage.eliminados)
	}
}

func TestEditarPerfil_NombreLargo(t *testing.T) {
	_, _, svc := nuevoServicioIntegrante(nil)

	// Se cuentan caracteres, no bytes: 50 letras con tilde siguen siendo válidas.
	if _, _, err := svc.EditarPerfil(1, strings.Repeat("á", LargoMaximoNombreIntegrante), nil, nil, false); err != nil {
		t.Fatalf("el nombre en el límite debería aceptarse: %v", err)
	}

	_, _, err := svc.EditarPerfil(1, strings.Repeat("a", LargoMaximoNombreIntegrante+1), nil, nil, false)

	if err != ErrPerfilNombreLargo {
		t.Fatalf("esperaba ErrPerfilNombreLargo, obtuve %v", err)
	}
}

// HU-PER-B01

func TestEditarPerfil_RecortaNombreYDescripcion(t *testing.T) {
	repo, _, svc := nuevoServicioIntegrante(nil)

	integrante, _, err := svc.EditarPerfil(1, "  Ana  ", texto("  Bajista  "), nil, false)

	if err != nil {
		t.Fatal(err)
	}
	if repo.nombreGuardado != "Ana" || repo.descripcionGuardada == nil || *repo.descripcionGuardada != "Bajista" {
		t.Fatalf("guardado = %q / %v, quería \"Ana\" / \"Bajista\"", repo.nombreGuardado, repo.descripcionGuardada)
	}
	if integrante.DescripcionIntegrante == nil || *integrante.DescripcionIntegrante != "Bajista" {
		t.Fatalf("respuesta = %v, quería \"Bajista\"", integrante.DescripcionIntegrante)
	}
}

func TestEditarPerfil_DescripcionEnBlancoQuedaVacia(t *testing.T) {
	repo, _, svc := nuevoServicioIntegrante(nil)

	if _, _, err := svc.EditarPerfil(1, "Ana", texto("   "), nil, false); err != nil {
		t.Fatal(err)
	}
	if repo.descripcionGuardada != nil {
		t.Fatalf("descripción guardada = %q, quería NULL", *repo.descripcionGuardada)
	}
}

func TestEditarPerfil_NombreVacio(t *testing.T) {
	repo, _, svc := nuevoServicioIntegrante(nil)

	if _, _, err := svc.EditarPerfil(1, "   ", nil, nil, false); err != ErrPerfilNombreObligatorio {
		t.Fatalf("err = %v, quería ErrPerfilNombreObligatorio", err)
	}
	if repo.actualizado {
		t.Fatal("no debería guardar")
	}
}

func TestEditarPerfil_ValidaElAvatar(t *testing.T) {
	casos := []struct {
		nombre  string
		archivo ArchivoImagen
		quiero  error
	}{
		{"gif", ArchivoImagen{Contenido: strings.NewReader("x"), NombreOriginal: "a.gif", Tamano: 10}, ErrPerfilAvatarFormatoInvalido},
		{"sin extensión", ArchivoImagen{Contenido: strings.NewReader("x"), NombreOriginal: "avatar", Tamano: 10}, ErrPerfilAvatarFormatoInvalido},
		{"más de 5 MB", ArchivoImagen{Contenido: strings.NewReader("x"), NombreOriginal: "a.png", Tamano: TamanoMaximoAvatar + 1}, ErrPerfilAvatarDemasiadoGrande},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			repo, storage, svc := nuevoServicioIntegrante(nil)

			if _, _, err := svc.EditarPerfil(1, "Ana", nil, &caso.archivo, false); err != caso.quiero {
				t.Fatalf("err = %v, quería %v", err, caso.quiero)
			}
			if repo.actualizado || len(storage.subidos) != 0 {
				t.Fatal("no debería subir ni guardar")
			}
		})
	}
}

func TestEditarPerfil_AceptaJPGPNGYWEBP(t *testing.T) {
	for _, extension := range []string{"jpg", "jpeg", "PNG", "webp"} {
		t.Run(extension, func(t *testing.T) {
			repo, storage, svc := nuevoServicioIntegrante(nil)
			archivo := ArchivoImagen{Contenido: strings.NewReader("x"), NombreOriginal: "foto." + extension, Tamano: TamanoMaximoAvatar}

			_, avatarUrl, err := svc.EditarPerfil(1, "Ana", nil, &archivo, false)

			if err != nil {
				t.Fatal(err)
			}
			if len(storage.subidos) != 1 || repo.avatarGuardado == nil || avatarUrl == nil {
				t.Fatalf("subidos = %v, guardado = %v, url = %v", storage.subidos, repo.avatarGuardado, avatarUrl)
			}
		})
	}
}

func TestEditarPerfil_FallaElAlmacenamiento(t *testing.T) {
	repo, storage, svc := nuevoServicioIntegrante(nil)
	storage.errSubir = errors.New("minio caído")
	archivo := ArchivoImagen{Contenido: strings.NewReader("x"), NombreOriginal: "a.png", Tamano: 10}

	if _, _, err := svc.EditarPerfil(1, "Ana", nil, &archivo, false); err != ErrPerfilErrorAlmacenamiento {
		t.Fatalf("err = %v, quería ErrPerfilErrorAlmacenamiento", err)
	}
	if repo.actualizado {
		t.Fatal("no debería guardar el perfil si no se pudo subir el avatar")
	}
}
