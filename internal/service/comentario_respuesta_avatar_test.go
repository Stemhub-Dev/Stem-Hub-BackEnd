package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
)

// HU-VER-F10: el listado de comentarios trae el avatar del autor (del
// comentario y de cada respuesta), y una respuesta se puede eliminar con las
// mismas reglas que un comentario.

type comentarioRespuestaFalso struct {
	repository.ComentarioRepository

	comentarios []dto.ComentarioListadoResponse

	codigoAutorRespuesta int64
	errAutorRespuesta    error
	bajaRespuesta        int64
}

func (r *comentarioRespuestaFalso) ListarPorVersion(int64, int64) ([]dto.ComentarioListadoResponse, error) {
	return r.comentarios, nil
}

func (r *comentarioRespuestaFalso) ObtenerAutorRespuesta(int64, int64, int64) (int64, error) {
	return r.codigoAutorRespuesta, r.errAutorRespuesta
}

func (r *comentarioRespuestaFalso) DarDeBajaRespuesta(codigoRespuesta int64) (bool, error) {
	r.bajaRespuesta = codigoRespuesta
	return true, nil
}

// Cuenta las firmas para verificar que un mismo avatar se firma una sola vez.
type storageContador struct {
	storageFalso
	firmas int
}

func (s *storageContador) ObtenerURLDescarga(ctx context.Context, key string, vigencia time.Duration) (string, error) {
	s.firmas++
	return s.storageFalso.ObtenerURLDescarga(ctx, key, vigencia)
}

func nuevoServicioRespuestas(
	repo *comentarioRespuestaFalso,
	proyectos *proyectoAccesoFalso,
	almacenamiento storage.AudioStorage,
) ComentarioService {
	return NewComentarioService(
		repo,
		proyectos,
		&cancionExistenciaFalsa{},
		// El usuario autenticado es el integrante 42.
		&integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 42}},
		nil,
		almacenamiento,
	)
}

func TestListarComentarios_ResuelveAvataresDeAutoresYRespuestas(t *testing.T) {
	repo := &comentarioRespuestaFalso{
		comentarios: []dto.ComentarioListadoResponse{
			{
				CodigoComentario: 1,
				Autor:            dto.AutorComentarioResponse{CodigoIntegrante: 7, AvatarObjectKey: texto("avatares/7.png")},
				Respuestas: []dto.RespuestaComentarioResponse{
					{CodigoRespuesta: 10, Autor: dto.AutorComentarioResponse{CodigoIntegrante: 8}},
					{CodigoRespuesta: 11, Autor: dto.AutorComentarioResponse{CodigoIntegrante: 7, AvatarObjectKey: texto("avatares/7.png")}},
				},
			},
			{
				CodigoComentario: 2,
				Autor:            dto.AutorComentarioResponse{CodigoIntegrante: 7, AvatarObjectKey: texto("avatares/7.png")},
			},
		},
	}
	almacenamiento := &storageContador{}

	comentarios, err := nuevoServicioRespuestas(repo, &proyectoAccesoFalso{}, almacenamiento).
		ListarPorVersion(1, 7, 8, 9)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	esperada := "https://storage.test/avatares/7.png"
	for _, autor := range []dto.AutorComentarioResponse{
		comentarios[0].Autor,
		comentarios[0].Respuestas[1].Autor,
		comentarios[1].Autor,
	} {
		if autor.AvatarUrl == nil || *autor.AvatarUrl != esperada {
			t.Fatalf("avatarUrl = %v, se esperaba %q", autor.AvatarUrl, esperada)
		}
	}
	if comentarios[0].Respuestas[0].Autor.AvatarUrl != nil {
		t.Fatal("el integrante sin avatar no debería tener avatarUrl")
	}
	if almacenamiento.firmas != 1 {
		t.Fatalf("firmas = %d, se esperaba 1 por avatar distinto", almacenamiento.firmas)
	}
}

func TestListarComentarios_FallaSiNoPuedeFirmarElAvatar(t *testing.T) {
	repo := &comentarioRespuestaFalso{
		comentarios: []dto.ComentarioListadoResponse{
			{Autor: dto.AutorComentarioResponse{AvatarObjectKey: texto("avatares/7.png")}},
		},
	}

	_, err := nuevoServicioRespuestas(
		repo,
		&proyectoAccesoFalso{},
		storageFalso{err: errors.New("minio caído")},
	).ListarPorVersion(1, 7, 8, 9)

	if err == nil {
		t.Fatal("se esperaba el error del storage")
	}
}

func TestEliminarRespuesta_PropiaNoPideElPermiso(t *testing.T) {
	repo := &comentarioRespuestaFalso{codigoAutorRespuesta: 42}
	proyectos := &proyectoAccesoFalso{sinPermiso: true}

	err := nuevoServicioRespuestas(repo, proyectos, nil).EliminarRespuesta(1, 7, 8, 9, 10, 20)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if proyectos.permisoPide != "" {
		t.Fatalf("no debería consultar permisos, consultó %q", proyectos.permisoPide)
	}
	if repo.bajaRespuesta != 20 {
		t.Fatalf("respuesta dada de baja = %d, se esperaba 20", repo.bajaRespuesta)
	}
}

func TestEliminarRespuesta_AjenaSinPermisoDevuelveSinPermiso(t *testing.T) {
	repo := &comentarioRespuestaFalso{codigoAutorRespuesta: 99}
	proyectos := &proyectoAccesoFalso{sinPermiso: true}

	err := nuevoServicioRespuestas(repo, proyectos, nil).EliminarRespuesta(1, 7, 8, 9, 10, 20)

	if !errors.Is(err, ErrComentarioSinPermisoEliminar) {
		t.Fatalf("err = %v, se esperaba ErrComentarioSinPermisoEliminar", err)
	}
	if proyectos.permisoPide != "GESTIONAR_COMENTARIOS" {
		t.Fatalf("permiso consultado = %q", proyectos.permisoPide)
	}
	if repo.bajaRespuesta != 0 {
		t.Fatal("no debería dar de baja la respuesta ajena")
	}
}

func TestEliminarRespuesta_AjenaConPermisoLaElimina(t *testing.T) {
	repo := &comentarioRespuestaFalso{codigoAutorRespuesta: 99}

	err := nuevoServicioRespuestas(repo, &proyectoAccesoFalso{}, nil).EliminarRespuesta(1, 7, 8, 9, 10, 20)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.bajaRespuesta != 20 {
		t.Fatalf("respuesta dada de baja = %d, se esperaba 20", repo.bajaRespuesta)
	}
}

func TestEliminarRespuesta_InexistenteDevuelveNoEncontrada(t *testing.T) {
	repo := &comentarioRespuestaFalso{errAutorRespuesta: sql.ErrNoRows}

	err := nuevoServicioRespuestas(repo, &proyectoAccesoFalso{}, nil).EliminarRespuesta(1, 7, 8, 9, 10, 20)

	if !errors.Is(err, ErrRespuestaComentarioNoEncontrada) {
		t.Fatalf("err = %v, se esperaba ErrRespuestaComentarioNoEncontrada", err)
	}
}
