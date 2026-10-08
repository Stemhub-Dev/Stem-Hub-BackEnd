package service

import (
	"strings"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

type perfilRepositoryFalso struct {
	repository.IntegranteRepository

	integrante model.Integrante

	avatarGuardado *string
}

func (r *perfilRepositoryFalso) BuscarPorCodigoUsuario(int64) (*model.Integrante, error) {
	copia := r.integrante
	return &copia, nil
}

func (r *perfilRepositoryFalso) ActualizarPerfil(_ int64, _ string, _ *string, avatar *string) error {
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
