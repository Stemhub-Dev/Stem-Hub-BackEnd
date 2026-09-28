package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// proyectoServiceFalso implementa solo ListarProyectos; el resto de la
// interfaz queda en el embebido nil.
type proyectoServiceFalso struct {
	service.ProyectoService

	llamado       bool
	codigoUsuario int64
	filtro        dto.ListarProyectosFiltro

	respuesta *dto.ProyectosPaginadosResponse
	err       error
}

func (s *proyectoServiceFalso) ListarProyectos(
	codigoUsuario int64,
	filtro dto.ListarProyectosFiltro,
) (*dto.ProyectosPaginadosResponse, error) {

	s.llamado = true
	s.codigoUsuario = codigoUsuario
	s.filtro = filtro

	return s.respuesta, s.err
}

func paginaDeProyectosVacia() *dto.ProyectosPaginadosResponse {
	return &dto.ProyectosPaginadosResponse{
		Data:        []dto.ProyectoListadoResponse{},
		CurrentPage: 1,
	}
}

func ejecutarListarProyectos(
	t *testing.T,
	servicio *proyectoServiceFalso,
	usuario *model.Usuario,
	url string,
) *httptest.ResponseRecorder {

	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/proyectos", func(c *gin.Context) {
		if usuario != nil {
			c.Set(middleware.UsuarioContextKey, usuario)
		}
		c.Next()
	}, NewProyectoHandler(servicio).Listar)

	grabador := httptest.NewRecorder()
	router.ServeHTTP(grabador, httptest.NewRequest(http.MethodGet, url, nil))

	return grabador
}

func TestListarProyectos_SinUsuarioResponde401(t *testing.T) {
	servicio := &proyectoServiceFalso{}

	grabador := ejecutarListarProyectos(t, servicio, nil, "/proyectos")

	if grabador.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba 401", grabador.Code)
	}
	if servicio.llamado {
		t.Fatal("no se debía llamar al service sin usuario")
	}
}

func TestListarProyectos_SinParametrosUsaValoresPorDefecto(t *testing.T) {
	servicio := &proyectoServiceFalso{respuesta: paginaDeProyectosVacia()}

	grabador := ejecutarListarProyectos(t, servicio, &model.Usuario{CodigoUsuario: 7}, "/proyectos")

	if grabador.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", grabador.Code)
	}

	esperado := dto.ListarProyectosFiltro{Pagina: 1, TamanoPagina: 10}
	if !reflect.DeepEqual(servicio.filtro, esperado) {
		t.Fatalf("filtro = %+v, se esperaba %+v", servicio.filtro, esperado)
	}
	if servicio.codigoUsuario != 7 {
		t.Fatalf("codigoUsuario = %d, se esperaba 7", servicio.codigoUsuario)
	}
}

func TestListarProyectos_PasaBusquedaFiltrosYPaginacion(t *testing.T) {
	servicio := &proyectoServiceFalso{respuesta: paginaDeProyectosVacia()}

	ejecutarListarProyectos(
		t,
		servicio,
		&model.Usuario{CodigoUsuario: 7},
		"/proyectos?q=%20Rock%20&estadoId=2&estadoId=3&tipoId=1&page=2&pageSize=25",
	)

	esperado := dto.ListarProyectosFiltro{
		Busqueda:     "Rock",
		Estados:      []int64{2, 3},
		Tipos:        []int64{1},
		Pagina:       2,
		TamanoPagina: 25,
	}
	if !reflect.DeepEqual(servicio.filtro, esperado) {
		t.Fatalf("filtro = %+v, se esperaba %+v", servicio.filtro, esperado)
	}
}

func TestListarProyectos_ParametrosInvalidosResponden400(t *testing.T) {
	casos := []string{
		"/proyectos?page=0",
		"/proyectos?page=abc",
		"/proyectos?pageSize=0",
		"/proyectos?pageSize=101",
		"/proyectos?estadoId=0",
		"/proyectos?estadoId=abc",
		"/proyectos?estadoId=2&estadoId=-1",
		"/proyectos?tipoId=0",
		"/proyectos?tipoId=Single",
		// Los filtros son por código: el nombre del catálogo no se acepta.
		"/proyectos?estadoId=En%20Desarrollo",
	}

	for _, url := range casos {
		t.Run(url, func(t *testing.T) {
			servicio := &proyectoServiceFalso{respuesta: paginaDeProyectosVacia()}

			grabador := ejecutarListarProyectos(t, servicio, &model.Usuario{CodigoUsuario: 7}, url)

			if grabador.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, se esperaba 400", grabador.Code)
			}
			if servicio.llamado {
				t.Fatal("no se debía llamar al service con parámetros inválidos")
			}

			var cuerpo map[string]string
			if err := json.Unmarshal(grabador.Body.Bytes(), &cuerpo); err != nil || cuerpo["error"] == "" {
				t.Fatalf("se esperaba un mensaje de error: %s", grabador.Body.String())
			}
		})
	}
}

func TestListarProyectos_DemasiadosCodigosResponde400(t *testing.T) {
	for _, parametro := range []string{"estadoId", "tipoId"} {
		t.Run(parametro, func(t *testing.T) {
			servicio := &proyectoServiceFalso{respuesta: paginaDeProyectosVacia()}

			valores := make([]string, 0, MaximoCodigosFiltro+1)
			for i := 0; i <= MaximoCodigosFiltro; i++ {
				valores = append(valores, parametro+"=1")
			}

			grabador := ejecutarListarProyectos(
				t,
				servicio,
				&model.Usuario{CodigoUsuario: 7},
				"/proyectos?"+strings.Join(valores, "&"),
			)

			if grabador.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, se esperaba 400", grabador.Code)
			}
		})
	}
}

func TestListarProyectos_ErrorInesperadoResponde500(t *testing.T) {
	servicio := &proyectoServiceFalso{err: errors.New("fallo de base")}

	grabador := ejecutarListarProyectos(t, servicio, &model.Usuario{CodigoUsuario: 7}, "/proyectos")

	if grabador.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, se esperaba 500", grabador.Code)
	}
}

func TestListarProyectos_FormaDeLaRespuesta(t *testing.T) {
	portada := "https://storage.test/logo.png"
	servicio := &proyectoServiceFalso{
		respuesta: &dto.ProyectosPaginadosResponse{
			Data: []dto.ProyectoListadoResponse{{
				CodigoProyecto:          9,
				Nombre:                  "Rock Nacional",
				Tipo:                    "Single",
				Estado:                  "En Desarrollo",
				CantidadCanciones:       3,
				CodRol:                  1,
				NombreRol:               "Productor",
				EsPropietario:           true,
				FechaUltimaModificacion: time.Date(2026, 9, 20, 15, 30, 0, 0, time.UTC),
				PortadaURL:              &portada,
			}},
			TotalItems:  11,
			TotalPages:  2,
			CurrentPage: 2,
		},
	}

	grabador := ejecutarListarProyectos(t, servicio, &model.Usuario{CodigoUsuario: 7}, "/proyectos?page=2")

	if grabador.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", grabador.Code)
	}

	var cuerpo map[string]any
	if err := json.Unmarshal(grabador.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}

	if cuerpo["totalItems"].(float64) != 11 ||
		cuerpo["totalPages"].(float64) != 2 ||
		cuerpo["currentPage"].(float64) != 2 {
		t.Fatalf("metadatos de paginación incorrectos: %s", grabador.Body.String())
	}

	item := cuerpo["data"].([]any)[0].(map[string]any)

	// Contrato que ya consumen las pantallas, más los dos campos nuevos.
	for _, campo := range []string{
		"codigoProyecto", "nombre", "descripcion", "logo", "tipo", "estado",
		"cantidadCanciones", "codRol", "nombreRol", "esPropietario",
		"fechaUltimaModificacion", "portadaUrl",
	} {
		if _, existe := item[campo]; !existe {
			t.Fatalf("falta el campo %q en %v", campo, item)
		}
	}

	if item["fechaUltimaModificacion"] != "2026-09-20T15:30:00Z" {
		t.Fatalf("fechaUltimaModificacion = %v, se esperaba RFC 3339", item["fechaUltimaModificacion"])
	}
	if item["portadaUrl"] != portada {
		t.Fatalf("portadaUrl = %v", item["portadaUrl"])
	}
}
