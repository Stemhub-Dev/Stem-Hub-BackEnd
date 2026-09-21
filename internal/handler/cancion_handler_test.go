package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// cancionServiceFalso implementa solo ListarMisCanciones; el resto de los
// métodos de la interfaz quedan en el embebido nil y no se usan en estos tests.
type cancionServiceFalso struct {
	service.CancionService

	llamado       bool
	codigoUsuario int64
	filtro        dto.ListarMisCancionesFiltro

	respuesta *dto.MisCancionesPaginadasResponse
	err       error
}

func (s *cancionServiceFalso) ListarMisCanciones(
	codigoUsuario int64,
	filtro dto.ListarMisCancionesFiltro,
) (*dto.MisCancionesPaginadasResponse, error) {

	s.llamado = true
	s.codigoUsuario = codigoUsuario
	s.filtro = filtro

	return s.respuesta, s.err
}

func ejecutarListarMisCanciones(
	t *testing.T,
	servicio *cancionServiceFalso,
	usuario *model.Usuario,
	url string,
) *httptest.ResponseRecorder {

	t.Helper()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/canciones", func(c *gin.Context) {
		if usuario != nil {
			c.Set(middleware.UsuarioContextKey, usuario)
		}
		c.Next()
	}, NewCancionHandler(servicio).ListarMisCanciones)

	grabador := httptest.NewRecorder()
	router.ServeHTTP(
		grabador,
		httptest.NewRequest(http.MethodGet, url, nil),
	)

	return grabador
}

func respuestaVacia() *dto.MisCancionesPaginadasResponse {
	return &dto.MisCancionesPaginadasResponse{
		Data:        []dto.MiCancionListadoResponse{},
		CurrentPage: 1,
	}
}

func TestListarMisCanciones_SinUsuarioResponde401(t *testing.T) {
	servicio := &cancionServiceFalso{}

	grabador := ejecutarListarMisCanciones(t, servicio, nil, "/canciones")

	if grabador.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba 401", grabador.Code)
	}
	if servicio.llamado {
		t.Fatal("no se debía llamar al service sin usuario")
	}
}

func TestListarMisCanciones_SinParametrosUsaValoresPorDefecto(t *testing.T) {
	servicio := &cancionServiceFalso{respuesta: respuestaVacia()}

	grabador := ejecutarListarMisCanciones(
		t,
		servicio,
		&model.Usuario{CodigoUsuario: 7},
		"/canciones",
	)

	if grabador.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", grabador.Code)
	}

	esperado := dto.ListarMisCancionesFiltro{
		Busqueda:     "",
		Proyectos:    nil,
		Orden:        dto.OrdenMisCancionesReciente,
		Pagina:       1,
		TamanoPagina: 10,
	}
	if !reflect.DeepEqual(servicio.filtro, esperado) {
		t.Fatalf("filtro = %+v, se esperaba %+v", servicio.filtro, esperado)
	}
	if servicio.codigoUsuario != 7 {
		t.Fatalf("codigoUsuario = %d, se esperaba 7", servicio.codigoUsuario)
	}
}

func TestListarMisCanciones_PasaBusquedaFiltrosYPaginacion(t *testing.T) {
	servicio := &cancionServiceFalso{respuesta: respuestaVacia()}

	ejecutarListarMisCanciones(
		t,
		servicio,
		&model.Usuario{CodigoUsuario: 7},
		"/canciones?q=%20Balada%20&proyectoId=9&proyectoId=4&sort=nombreAsc&page=2&pageSize=25",
	)

	esperado := dto.ListarMisCancionesFiltro{
		Busqueda:     "Balada",
		Proyectos:    []int64{9, 4},
		Orden:        dto.OrdenMisCancionesNombreAsc,
		Pagina:       2,
		TamanoPagina: 25,
	}
	if !reflect.DeepEqual(servicio.filtro, esperado) {
		t.Fatalf("filtro = %+v, se esperaba %+v", servicio.filtro, esperado)
	}
}

func TestListarMisCanciones_OrdenesAdmitidos(t *testing.T) {
	for _, orden := range []string{
		dto.OrdenMisCancionesReciente,
		dto.OrdenMisCancionesNombreAsc,
		dto.OrdenMisCancionesNombreDesc,
	} {
		t.Run(orden, func(t *testing.T) {
			servicio := &cancionServiceFalso{respuesta: respuestaVacia()}

			grabador := ejecutarListarMisCanciones(
				t,
				servicio,
				&model.Usuario{CodigoUsuario: 7},
				"/canciones?sort="+orden,
			)

			if grabador.Code != http.StatusOK {
				t.Fatalf("status = %d, se esperaba 200", grabador.Code)
			}
			if servicio.filtro.Orden != orden {
				t.Fatalf("orden = %q, se esperaba %q", servicio.filtro.Orden, orden)
			}
		})
	}
}

func TestListarMisCanciones_ParametrosInvalidosResponden400(t *testing.T) {
	casos := []string{
		"/canciones?page=0",
		"/canciones?page=-1",
		"/canciones?page=abc",
		"/canciones?pageSize=0",
		"/canciones?pageSize=abc",
		"/canciones?pageSize=101",
		"/canciones?page=1.5",
		"/canciones?sort=fechaAlta",
		"/canciones?sort=c.nombrecancion",
		"/canciones?proyectoId=0",
		"/canciones?proyectoId=-3",
		"/canciones?proyectoId=abc",
		"/canciones?proyectoId=9&proyectoId=abc",
	}

	for _, url := range casos {
		t.Run(url, func(t *testing.T) {
			servicio := &cancionServiceFalso{respuesta: respuestaVacia()}

			grabador := ejecutarListarMisCanciones(
				t,
				servicio,
				&model.Usuario{CodigoUsuario: 7},
				url,
			)

			if grabador.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, se esperaba 400", grabador.Code)
			}
			if servicio.llamado {
				t.Fatal("no se debía llamar al service con parámetros inválidos")
			}
		})
	}
}

func TestListarMisCanciones_DemasiadosProyectosResponde400(t *testing.T) {
	servicio := &cancionServiceFalso{respuesta: respuestaVacia()}

	valores := make([]string, 0, MaximoCodigosFiltro+1)
	for i := 0; i <= MaximoCodigosFiltro; i++ {
		valores = append(valores, "proyectoId=1")
	}

	grabador := ejecutarListarMisCanciones(
		t,
		servicio,
		&model.Usuario{CodigoUsuario: 7},
		"/canciones?"+strings.Join(valores, "&"),
	)

	if grabador.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, se esperaba 400", grabador.Code)
	}
	if servicio.llamado {
		t.Fatal("no se debía llamar al service con demasiados proyectos")
	}
}

func TestListarMisCanciones_PageSizeMaximoEsValido(t *testing.T) {
	servicio := &cancionServiceFalso{respuesta: respuestaVacia()}

	grabador := ejecutarListarMisCanciones(
		t,
		servicio,
		&model.Usuario{CodigoUsuario: 7},
		"/canciones?pageSize=100",
	)

	if grabador.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", grabador.Code)
	}
}

func TestListarMisCanciones_PerfilRequeridoResponde403(t *testing.T) {
	servicio := &cancionServiceFalso{err: service.ErrCancionPerfilRequerido}

	grabador := ejecutarListarMisCanciones(
		t,
		servicio,
		&model.Usuario{CodigoUsuario: 7},
		"/canciones",
	)

	if grabador.Code != http.StatusForbidden {
		t.Fatalf("status = %d, se esperaba 403", grabador.Code)
	}
}

func TestListarMisCanciones_ErrorInesperadoResponde500(t *testing.T) {
	servicio := &cancionServiceFalso{err: errors.New("fallo de base")}

	grabador := ejecutarListarMisCanciones(
		t,
		servicio,
		&model.Usuario{CodigoUsuario: 7},
		"/canciones",
	)

	if grabador.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, se esperaba 500", grabador.Code)
	}
}

func TestListarMisCanciones_FormaDeLaRespuesta(t *testing.T) {
	servicio := &cancionServiceFalso{
		respuesta: &dto.MisCancionesPaginadasResponse{
			Data: []dto.MiCancionListadoResponse{
				{
					CodigoCancion:  3,
					Nombre:         "Balada",
					CodigoProyecto: 9,
					NombreProyecto: "Disco",
				},
			},
			TotalItems:  11,
			TotalPages:  2,
			CurrentPage: 2,
		},
	}

	grabador := ejecutarListarMisCanciones(
		t,
		servicio,
		&model.Usuario{CodigoUsuario: 7},
		"/canciones?page=2",
	)

	if grabador.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", grabador.Code)
	}

	var cuerpo map[string]any
	if err := json.Unmarshal(grabador.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}

	for _, campo := range []string{"data", "totalItems", "totalPages", "currentPage"} {
		if _, existe := cuerpo[campo]; !existe {
			t.Fatalf("falta el campo %q en %s", campo, grabador.Body.String())
		}
	}

	items := cuerpo["data"].([]any)
	item := items[0].(map[string]any)

	// Contrato consumido por el frontend (MiCancionListado).
	for _, campo := range []string{"codigoCancion", "nombre", "codigoProyecto", "nombreProyecto", "versionActual"} {
		if _, existe := item[campo]; !existe {
			t.Fatalf("falta el campo %q en el ítem %v", campo, item)
		}
	}

	if cuerpo["totalItems"].(float64) != 11 ||
		cuerpo["totalPages"].(float64) != 2 ||
		cuerpo["currentPage"].(float64) != 2 {
		t.Fatalf("metadatos de paginación incorrectos: %s", grabador.Body.String())
	}
}
