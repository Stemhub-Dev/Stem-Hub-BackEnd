package service

import (
	"errors"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

type comentarioRepositoryFalso struct {
	repository.ComentarioRepository

	stemExiste   bool
	stemCreado   *int64
	crearLlamado bool
}

func (r *comentarioRepositoryFalso) ExisteStemActivoEnVersion(int64, int64) (bool, error) {
	return r.stemExiste, nil
}

func (r *comentarioRepositoryFalso) Crear(
	_ int64,
	_ int64,
	codStem *int64,
	_ string,
	_ *float64,
	_ *float64,
) (int64, error) {
	r.crearLlamado = true
	r.stemCreado = codStem
	return 1, nil
}

func nuevoServicioComentarios(repo *comentarioRepositoryFalso) ComentarioService {
	return NewComentarioService(
		repo,
		&proyectoAccesoFalso{},
		&cancionExistenciaFalsa{},
		&integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 42}},
		nil,
	)
}

func TestCrearComentario_DeUnStem(t *testing.T) {
	repo := &comentarioRepositoryFalso{stemExiste: true}

	respuesta, err := nuevoServicioComentarios(repo).Crear(1, 7, 8, 9,
		dto.CrearComentarioRequest{Texto: "El bombo satura", CodStem: codigo(10)},
	)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.stemCreado == nil || *repo.stemCreado != 10 {
		t.Fatalf("codStem guardado = %v", repo.stemCreado)
	}
	if respuesta.CodStem == nil || *respuesta.CodStem != 10 {
		t.Fatalf("codStem respondido = %v", respuesta.CodStem)
	}
}

func TestCrearComentario_StemDeOtraVersion(t *testing.T) {
	repo := &comentarioRepositoryFalso{stemExiste: false}

	_, err := nuevoServicioComentarios(repo).Crear(1, 7, 8, 9,
		dto.CrearComentarioRequest{Texto: "Hola", CodStem: codigo(10)},
	)

	if !errors.Is(err, ErrComentarioStemNoEncontrado) {
		t.Fatalf("err = %v", err)
	}
	if repo.crearLlamado {
		t.Fatal("no debería crear el comentario")
	}
}

func TestCrearComentario_DeLaVersionNoLlevaStem(t *testing.T) {
	repo := &comentarioRepositoryFalso{}

	if _, err := nuevoServicioComentarios(repo).Crear(1, 7, 8, 9,
		dto.CrearComentarioRequest{Texto: "Hola"},
	); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.stemCreado != nil {
		t.Fatalf("codStem = %v, se esperaba nil", *repo.stemCreado)
	}
}
