package service

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// Los falsos embeben la interfaz real (nil) e implementan solo los métodos
// que usa ListarMisCanciones.

type integranteRepositoryFalso struct {
	repository.IntegranteRepository

	integrante *model.Integrante
	err        error
}

func (r *integranteRepositoryFalso) BuscarPorCodigoUsuario(
	int64,
) (*model.Integrante, error) {
	return r.integrante, r.err
}

type cancionRepositoryFalso struct {
	repository.CancionRepository

	total    int
	errTotal error

	canciones    []dto.MiCancionListadoResponse
	errCanciones error

	listarLlamado  bool
	busquedaContar string
	busquedaListar string
	limite         int
	desplazamiento int
}

func (r *cancionRepositoryFalso) ContarPorIntegrante(
	_ int64,
	busqueda string,
) (int, error) {
	r.busquedaContar = busqueda
	return r.total, r.errTotal
}

func (r *cancionRepositoryFalso) ListarPorIntegrante(
	_ int64,
	busqueda string,
	limite int,
	desplazamiento int,
) ([]dto.MiCancionListadoResponse, error) {
	r.listarLlamado = true
	r.busquedaListar = busqueda
	r.limite = limite
	r.desplazamiento = desplazamiento
	return r.canciones, r.errCanciones
}

func nuevoServicioMisCanciones(
	canciones *cancionRepositoryFalso,
) CancionService {
	return NewCancionService(
		canciones,
		nil,
		&integranteRepositoryFalso{
			integrante: &model.Integrante{CodIntegrante: 42},
		},
		nil,
	)
}

func filtro(pagina, tamano int) dto.ListarMisCancionesFiltro {
	return dto.ListarMisCancionesFiltro{
		Busqueda:     "bal",
		Pagina:       pagina,
		TamanoPagina: tamano,
	}
}

func TestListarMisCanciones_CalculaTotalPages(t *testing.T) {
	casos := []struct {
		nombre       string
		total        int
		tamano       int
		totalPaginas int
	}{
		{"sin resultados", 0, 10, 0},
		{"menos de una página", 3, 10, 1},
		{"divisible exacto", 20, 10, 2},
		{"con resto", 21, 10, 3},
		{"tamaño uno", 5, 1, 5},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &cancionRepositoryFalso{total: caso.total}

			resultado, err := nuevoServicioMisCanciones(repo).
				ListarMisCanciones(1, filtro(1, caso.tamano))

			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if resultado.TotalPages != caso.totalPaginas {
				t.Fatalf("totalPages = %d, se esperaba %d", resultado.TotalPages, caso.totalPaginas)
			}
			if resultado.TotalItems != caso.total {
				t.Fatalf("totalItems = %d, se esperaba %d", resultado.TotalItems, caso.total)
			}
		})
	}
}

func TestListarMisCanciones_CalculaDesplazamientoYPasaBusqueda(t *testing.T) {
	repo := &cancionRepositoryFalso{
		total: 25,
		canciones: []dto.MiCancionListadoResponse{
			{CodigoCancion: 1},
			{CodigoCancion: 2},
		},
	}

	resultado, err := nuevoServicioMisCanciones(repo).
		ListarMisCanciones(1, filtro(2, 10))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.limite != 10 || repo.desplazamiento != 10 {
		t.Fatalf("limite/desplazamiento = %d/%d, se esperaba 10/10", repo.limite, repo.desplazamiento)
	}
	if repo.busquedaContar != "bal" || repo.busquedaListar != "bal" {
		t.Fatal("la búsqueda debe aplicarse tanto al conteo como al listado")
	}
	if resultado.CurrentPage != 2 || len(resultado.Data) != 2 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
}

func TestListarMisCanciones_PaginaFueraDeRangoDevuelveDataVacia(t *testing.T) {
	repo := &cancionRepositoryFalso{total: 15}

	resultado, err := nuevoServicioMisCanciones(repo).
		ListarMisCanciones(1, filtro(5, 10))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.listarLlamado {
		t.Fatal("no se debía consultar el listado para una página fuera de rango")
	}
	if resultado.Data == nil || len(resultado.Data) != 0 {
		t.Fatalf("data = %#v, se esperaba slice vacío no nil", resultado.Data)
	}
	if resultado.CurrentPage != 5 || resultado.TotalPages != 2 || resultado.TotalItems != 15 {
		t.Fatalf("metadatos incorrectos: %+v", resultado)
	}
}

func TestListarMisCanciones_PaginaEnormeNoDesborda(t *testing.T) {
	repo := &cancionRepositoryFalso{total: 3}

	resultado, err := nuevoServicioMisCanciones(repo).
		ListarMisCanciones(1, filtro(int(^uint(0)>>1), 100))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.listarLlamado || len(resultado.Data) != 0 {
		t.Fatal("una página enorme debe responder vacía sin consultar")
	}
}

func TestListarMisCanciones_RepositorioSinFilasDevuelveDataNoNil(t *testing.T) {
	repo := &cancionRepositoryFalso{total: 1, canciones: nil}

	resultado, err := nuevoServicioMisCanciones(repo).
		ListarMisCanciones(1, filtro(1, 10))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if resultado.Data == nil {
		t.Fatal("data no debe ser nil (se serializaría como null)")
	}
}

func TestListarMisCanciones_SinPerfilDevuelveErrCancionPerfilRequerido(t *testing.T) {
	servicio := NewCancionService(
		&cancionRepositoryFalso{},
		nil,
		&integranteRepositoryFalso{err: sql.ErrNoRows},
		nil,
	)

	_, err := servicio.ListarMisCanciones(1, filtro(1, 10))

	if !errors.Is(err, ErrCancionPerfilRequerido) {
		t.Fatalf("err = %v, se esperaba ErrCancionPerfilRequerido", err)
	}
}

func TestListarMisCanciones_PropagaErrores(t *testing.T) {
	errBase := errors.New("fallo de base")

	casos := []struct {
		nombre      string
		integrantes *integranteRepositoryFalso
		canciones   *cancionRepositoryFalso
	}{
		{
			"error buscando integrante",
			&integranteRepositoryFalso{err: errBase},
			&cancionRepositoryFalso{},
		},
		{
			"error contando",
			&integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 1}},
			&cancionRepositoryFalso{errTotal: errBase},
		},
		{
			"error listando",
			&integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 1}},
			&cancionRepositoryFalso{total: 5, errCanciones: errBase},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio := NewCancionService(caso.canciones, nil, caso.integrantes, nil)

			_, err := servicio.ListarMisCanciones(1, filtro(1, 10))

			if !errors.Is(err, errBase) {
				t.Fatalf("err = %v, se esperaba %v", err, errBase)
			}
		})
	}
}
