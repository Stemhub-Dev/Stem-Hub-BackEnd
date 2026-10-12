package service

import (
	"errors"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// PRU-09 (HU-COM-B04): un integrante sin GESTIONAR_COMENTARIOS (rol Músico)
// no puede eliminar el comentario de otro; el autor sí puede eliminar el suyo.

type comentarioAutorFalso struct {
	repository.ComentarioRepository

	codigoAutor    int64
	bajaLlamada    bool
	bajaComentario int64
}

func (r *comentarioAutorFalso) ObtenerAutorComentario(int64, int64) (int64, error) {
	return r.codigoAutor, nil
}

func (r *comentarioAutorFalso) DarDeBaja(codigoComentario int64) (bool, error) {
	r.bajaLlamada = true
	r.bajaComentario = codigoComentario
	return true, nil
}

func nuevoServicioEliminarComentario(
	repo *comentarioAutorFalso,
	proyectos *proyectoAccesoFalso,
) ComentarioService {
	return NewComentarioService(
		repo,
		proyectos,
		&cancionExistenciaFalsa{},
		// El usuario autenticado es el integrante 42.
		&integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 42}},
		nil,
	)
}

func TestEliminarComentario_AjenoSinPermisoDevuelveSinPermiso(t *testing.T) {
	repo := &comentarioAutorFalso{codigoAutor: 99}
	proyectos := &proyectoAccesoFalso{sinPermiso: true}

	err := nuevoServicioEliminarComentario(repo, proyectos).Eliminar(1, 7, 8, 9, 10)

	if !errors.Is(err, ErrComentarioSinPermisoEliminar) {
		t.Fatalf("err = %v, se esperaba ErrComentarioSinPermisoEliminar", err)
	}
	if proyectos.permisoPide != "GESTIONAR_COMENTARIOS" {
		t.Fatalf("permiso consultado = %q", proyectos.permisoPide)
	}
	if repo.bajaLlamada {
		t.Fatal("no debería dar de baja el comentario ajeno")
	}
}

func TestEliminarComentario_PropioNoPideElPermiso(t *testing.T) {
	repo := &comentarioAutorFalso{codigoAutor: 42}
	proyectos := &proyectoAccesoFalso{sinPermiso: true}

	if err := nuevoServicioEliminarComentario(repo, proyectos).Eliminar(1, 7, 8, 9, 10); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if proyectos.permisoPide != "" {
		t.Fatalf("no debería consultar permisos, consultó %q", proyectos.permisoPide)
	}
	if !repo.bajaLlamada || repo.bajaComentario != 10 {
		t.Fatalf("baja = %v (comentario %d)", repo.bajaLlamada, repo.bajaComentario)
	}
}

func TestEliminarComentario_AjenoConPermisoLoElimina(t *testing.T) {
	repo := &comentarioAutorFalso{codigoAutor: 99}

	if err := nuevoServicioEliminarComentario(repo, &proyectoAccesoFalso{}).Eliminar(1, 7, 8, 9, 10); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !repo.bajaLlamada {
		t.Fatal("debería dar de baja el comentario")
	}
}
