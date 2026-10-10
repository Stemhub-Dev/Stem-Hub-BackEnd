package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/gin-gonic/gin"
)

// nuevoRouterDePrueba arma el router real con handlers vacíos: alcanza para
// verificar el cableado de rutas y middlewares, porque las requests de estos
// tests se cortan en la autenticación antes de llegar a ningún handler.
func nuevoRouterDePrueba(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	return NewRouter(
		nil,
		&handler.RolHandler{},
		&handler.GeneroMusicalHandler{},
		&handler.TipoProyectoHandler{},
		&handler.EstadoProyectoHandler{},
		&handler.UsuarioHandler{},
		&handler.UsuarioAdministracionHandler{},
		&handler.IntegranteHandler{},
		&handler.ProyectoHandler{},
		&handler.InvitacionProyectoHandler{},
		&handler.CancionHandler{},
		&handler.ComentarioHandler{},
		&handler.StemHandler{},
		&handler.ReporteHandler{},
		&handler.TableroHandler{},
		&handler.PermisoHandler{},
		&handler.RolPermisoHandler{},
		&middleware.AuthMiddleware{},
		&middleware.PermisoMiddleware{},
		nil,
	)
}

func TestMisCanciones_RequiereAutenticacion(t *testing.T) {
	router := nuevoRouterDePrueba(t)

	for _, url := range []string{
		"/canciones",
		"/canciones?q=balada&proyectoId=1&sort=nombreAsc&page=2&pageSize=5",
	} {
		t.Run(url, func(t *testing.T) {
			grabador := httptest.NewRecorder()
			router.ServeHTTP(
				grabador,
				httptest.NewRequest(http.MethodGet, url, nil),
			)

			// 401 (y no 404) confirma que la ruta existe y que pasa primero
			// por ValidarJWT: sin token no se llega al handler.
			if grabador.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, se esperaba 401", grabador.Code)
			}
		})
	}
}

func TestMisProyectos_RequiereAutenticacion(t *testing.T) {
	router := nuevoRouterDePrueba(t)

	for _, url := range []string{
		"/proyectos",
		"/proyectos?q=rock&estadoId=2&tipoId=1&page=2&pageSize=5",
	} {
		t.Run(url, func(t *testing.T) {
			grabador := httptest.NewRecorder()
			router.ServeHTTP(
				grabador,
				httptest.NewRequest(http.MethodGet, url, nil),
			)

			if grabador.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, se esperaba 401", grabador.Code)
			}
		})
	}
}

func TestMisCanciones_SoloAceptaGet(t *testing.T) {
	router := nuevoRouterDePrueba(t)

	grabador := httptest.NewRecorder()
	router.ServeHTTP(
		grabador,
		httptest.NewRequest(http.MethodPost, "/canciones", nil),
	)

	if grabador.Code != http.StatusNotFound && grabador.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, se esperaba 404 o 405", grabador.Code)
	}
}

func TestStems_RequierenAutenticacion(t *testing.T) {
	router := nuevoRouterDePrueba(t)
	base := "/proyectos/1/canciones/2/versiones/3/stems"

	for _, ruta := range []struct{ metodo, url string }{
		{http.MethodGet, "/proyectos/1/categorias-stem"},
		{http.MethodGet, base},
		{http.MethodPost, base},
		{http.MethodPut, base + "/4"},
		{http.MethodDelete, base + "/4"},
		{http.MethodGet, base + "/4/audio"},
	} {
		t.Run(ruta.metodo+" "+ruta.url, func(t *testing.T) {
			grabador := httptest.NewRecorder()
			router.ServeHTTP(grabador, httptest.NewRequest(ruta.metodo, ruta.url, nil))

			if grabador.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, se esperaba 401", grabador.Code)
			}
		})
	}
}

func TestRutasDeIA_RequierenAutenticacion(t *testing.T) {
	router := nuevoRouterDePrueba(t)

	base := "/proyectos/1/canciones/2/versiones/3"

	for _, caso := range []struct {
		metodo string
		url    string
	}{
		{http.MethodPost, base + "/stems/separacion"},
		{http.MethodGet, base + "/stems/separacion"},
		{http.MethodGet, base + "/comentarios/resumen"},
	} {
		t.Run(caso.metodo+" "+caso.url, func(t *testing.T) {
			grabador := httptest.NewRecorder()
			router.ServeHTTP(
				grabador,
				httptest.NewRequest(caso.metodo, caso.url, nil),
			)

			if grabador.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, se esperaba 401", grabador.Code)
			}
		})
	}
}

// PRU-07 (HU-SEG-B01): sin header Authorization, las rutas de versiones y
// comentarios responden 401 desde ValidarJWT. Los handlers de este router
// tienen el service en nil: si alguno llegara a ejecutarse, el test
// fallaría con un panic en vez de un 401.
func TestVersionesYComentarios_RequierenAutenticacion(t *testing.T) {
	router := nuevoRouterDePrueba(t)

	base := "/proyectos/1/canciones/2/versiones"

	for _, caso := range []struct {
		metodo string
		url    string
	}{
		{http.MethodPost, base},
		{http.MethodGet, base},
		{http.MethodGet, base + "/3/audio"},
		{http.MethodGet, base + "/3/comentarios"},
		{http.MethodPost, base + "/3/comentarios"},
		{http.MethodDelete, base + "/3/comentarios/4"},
	} {
		t.Run(caso.metodo+" "+caso.url, func(t *testing.T) {
			grabador := httptest.NewRecorder()
			router.ServeHTTP(
				grabador,
				httptest.NewRequest(caso.metodo, caso.url, nil),
			)

			if grabador.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, se esperaba 401", grabador.Code)
			}
		})
	}
}

// HU-DASH-B01 #7 / HU-DASH-B02 #9: sin token, el Tablero responde 401.
func TestTablero_RequiereAutenticacion(t *testing.T) {
	router := nuevoRouterDePrueba(t)

	for _, url := range []string{
		"/tablero/indicadores",
		"/tablero/indicadores?proyectoId=1",
		"/tablero/graficos/versiones-por-proyecto",
		"/tablero/graficos/canciones-por-etapa?proyectoId=1",
		"/tablero/graficos/actividad?agrupacion=mensual",
	} {
		t.Run(url, func(t *testing.T) {
			grabador := httptest.NewRecorder()
			router.ServeHTTP(
				grabador,
				httptest.NewRequest(http.MethodGet, url, nil),
			)

			if grabador.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, se esperaba 401", grabador.Code)
			}
		})
	}
}
