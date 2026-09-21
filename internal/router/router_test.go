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
