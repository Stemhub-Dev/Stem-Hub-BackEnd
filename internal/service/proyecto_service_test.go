package service

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

type proyectoRepositoryFalso struct {
	repository.ProyectoRepository

	total    int
	errTotal error

	proyectos    []dto.ProyectoListadoResponse
	errProyectos error

	contarLlamado  bool
	listarLlamado  bool
	filtroContar   dto.ListarProyectosFiltro
	filtroListar   dto.ListarProyectosFiltro
	desplazamiento int
}

func (r *proyectoRepositoryFalso) ContarPorIntegrante(
	_ int64,
	filtro dto.ListarProyectosFiltro,
) (int, error) {
	r.contarLlamado = true
	r.filtroContar = filtro
	return r.total, r.errTotal
}

func (r *proyectoRepositoryFalso) ListarPorIntegrante(
	_ int64,
	filtro dto.ListarProyectosFiltro,
	desplazamiento int,
) ([]dto.ProyectoListadoResponse, error) {
	r.listarLlamado = true
	r.filtroListar = filtro
	r.desplazamiento = desplazamiento
	return r.proyectos, r.errProyectos
}

// storageFalso firma URLs predecibles, o falla si se le indica.
type storageFalso struct {
	err error
}

func (storageFalso) Subir(context.Context, string, io.Reader, int64, string) error {
	return nil
}

func (s storageFalso) ObtenerURLDescarga(_ context.Context, objectKey string, _ time.Duration) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "https://storage.test/" + objectKey, nil
}

func nuevoServicioProyectos(
	proyectos *proyectoRepositoryFalso,
	integrantes *integranteRepositoryFalso,
	almacenamiento storageFalso,
) ProyectoService {
	return NewProyectoService(proyectos, integrantes, almacenamiento)
}

func integranteActivo() *integranteRepositoryFalso {
	return &integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 42}}
}

func filtroProyectos(pagina, tamano int) dto.ListarProyectosFiltro {
	return dto.ListarProyectosFiltro{
		Busqueda:     "rock",
		Estados:      []int64{2},
		Tipos:        []int64{1, 3},
		Pagina:       pagina,
		TamanoPagina: tamano,
	}
}

func TestListarProyectos_CalculaTotalPages(t *testing.T) {
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
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &proyectoRepositoryFalso{total: caso.total}

			resultado, err := nuevoServicioProyectos(repo, integranteActivo(), storageFalso{}).
				ListarProyectos(1, filtroProyectos(1, caso.tamano))

			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if resultado.TotalPages != caso.totalPaginas || resultado.TotalItems != caso.total {
				t.Fatalf("totalItems/totalPages = %d/%d, se esperaba %d/%d",
					resultado.TotalItems, resultado.TotalPages, caso.total, caso.totalPaginas)
			}
		})
	}
}

func TestListarProyectos_MismoFiltroAlConteoYAlListado(t *testing.T) {
	repo := &proyectoRepositoryFalso{
		total:     25,
		proyectos: []dto.ProyectoListadoResponse{{CodigoProyecto: 1}},
	}

	resultado, err := nuevoServicioProyectos(repo, integranteActivo(), storageFalso{}).
		ListarProyectos(1, filtroProyectos(3, 10))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !reflect.DeepEqual(repo.filtroContar, repo.filtroListar) ||
		!reflect.DeepEqual(repo.filtroListar, filtroProyectos(3, 10)) {
		t.Fatalf("filtros: conteo %+v, listado %+v", repo.filtroContar, repo.filtroListar)
	}
	if repo.desplazamiento != 20 {
		t.Fatalf("desplazamiento = %d, se esperaba 20", repo.desplazamiento)
	}
	if resultado.CurrentPage != 3 || len(resultado.Data) != 1 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
}

func TestListarProyectos_PaginaFueraDeRangoNoConsultaElListado(t *testing.T) {
	repo := &proyectoRepositoryFalso{total: 15}

	resultado, err := nuevoServicioProyectos(repo, integranteActivo(), storageFalso{}).
		ListarProyectos(1, filtroProyectos(int(^uint(0)>>1), 10))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.listarLlamado {
		t.Fatal("no se debía consultar el listado para una página fuera de rango")
	}
	if resultado.Data == nil || len(resultado.Data) != 0 || resultado.TotalPages != 2 || resultado.TotalItems != 15 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
}

func TestListarProyectos_DataNuncaEsNil(t *testing.T) {
	repo := &proyectoRepositoryFalso{total: 1, proyectos: nil}

	resultado, err := nuevoServicioProyectos(repo, integranteActivo(), storageFalso{}).
		ListarProyectos(1, filtroProyectos(1, 10))

	if err != nil || resultado.Data == nil {
		t.Fatalf("data = %#v, err = %v; se esperaba slice no nil", resultado.Data, err)
	}
}

func TestListarProyectos_SinPerfilODadoDeBajaDevuelvePaginaVacia(t *testing.T) {
	baja := time.Now()

	casos := map[string]*integranteRepositoryFalso{
		"sin perfil":          {err: sql.ErrNoRows},
		"perfil dado de baja": {integrante: &model.Integrante{CodIntegrante: 42, FechaHoraBajaIntegrante: &baja}},
	}

	for nombre, integrantes := range casos {
		t.Run(nombre, func(t *testing.T) {
			repo := &proyectoRepositoryFalso{total: 5}

			resultado, err := nuevoServicioProyectos(repo, integrantes, storageFalso{}).
				ListarProyectos(1, filtroProyectos(2, 10))

			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if repo.contarLlamado || repo.listarLlamado {
				t.Fatal("sin perfil activo no se debía consultar")
			}
			if resultado.Data == nil || len(resultado.Data) != 0 ||
				resultado.TotalItems != 0 || resultado.TotalPages != 0 || resultado.CurrentPage != 2 {
				t.Fatalf("resultado inesperado: %+v", resultado)
			}
		})
	}
}

func TestListarProyectos_FirmaLaPortadaSoloConLogo(t *testing.T) {
	logo := "proyectos/9/logo.png"
	repo := &proyectoRepositoryFalso{
		total: 2,
		proyectos: []dto.ProyectoListadoResponse{
			{CodigoProyecto: 9, Logo: &logo},
			{CodigoProyecto: 4},
		},
	}

	resultado, err := nuevoServicioProyectos(repo, integranteActivo(), storageFalso{}).
		ListarProyectos(1, filtroProyectos(1, 10))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if portada := resultado.Data[0].PortadaURL; portada == nil || *portada != "https://storage.test/"+logo {
		t.Fatalf("portada = %v, se esperaba la URL firmada", portada)
	}
	if resultado.Data[1].PortadaURL != nil {
		t.Fatal("sin logo no debe haber portada")
	}
}

func TestListarProyectos_UnFalloDelStorageNoTumbaElListado(t *testing.T) {
	logo := "proyectos/9/logo.png"
	repo := &proyectoRepositoryFalso{
		total:     1,
		proyectos: []dto.ProyectoListadoResponse{{CodigoProyecto: 9, Logo: &logo}},
	}

	resultado, err := nuevoServicioProyectos(repo, integranteActivo(), storageFalso{err: errors.New("storage caído")}).
		ListarProyectos(1, filtroProyectos(1, 10))

	if err != nil {
		t.Fatalf("un fallo al firmar la portada no debe fallar el listado: %v", err)
	}
	if len(resultado.Data) != 1 || resultado.Data[0].PortadaURL != nil {
		t.Fatalf("se esperaba el proyecto sin portada: %+v", resultado.Data)
	}
}

func TestListarProyectos_PropagaErrores(t *testing.T) {
	errBase := errors.New("fallo de base")

	casos := []struct {
		nombre      string
		integrantes *integranteRepositoryFalso
		proyectos   *proyectoRepositoryFalso
	}{
		{"error buscando integrante", &integranteRepositoryFalso{err: errBase}, &proyectoRepositoryFalso{}},
		{"error contando", integranteActivo(), &proyectoRepositoryFalso{errTotal: errBase}},
		{"error listando", integranteActivo(), &proyectoRepositoryFalso{total: 5, errProyectos: errBase}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := nuevoServicioProyectos(caso.proyectos, caso.integrantes, storageFalso{}).
				ListarProyectos(1, filtroProyectos(1, 10))

			if !errors.Is(err, errBase) {
				t.Fatalf("err = %v, se esperaba %v", err, errBase)
			}
		})
	}
}
