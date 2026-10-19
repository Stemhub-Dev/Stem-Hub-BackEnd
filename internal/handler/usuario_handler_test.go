package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Valida el nombre igual que el service real: el handler no debe cortar
// antes con su propio mensaje.
type usuarioServiceFalso struct {
	llamado bool
}

func (s *usuarioServiceFalso) ObtenerUsuarioActivoPorIDAutenticacion(string) (*model.Usuario, error) {
	return nil, nil
}

func (s *usuarioServiceFalso) RegistrarUsuario(
	_ string,
	_ string,
	nombre string,
	_ *string,
) (*model.Usuario, *model.Integrante, bool, error) {
	s.llamado = true
	if strings.TrimSpace(nombre) == "" {
		return nil, nil, false, service.ErrNombreIntegranteObligatorio
	}
	return &model.Usuario{}, &model.Integrante{}, true, nil
}

func ejecutarRegistrar(t *testing.T, servicio *usuarioServiceFalso, cuerpo string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h := NewUsuarioHandler(servicio, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ClaimsContextKey, &middleware.SupabaseClaims{
			Email:            "ana@example.com",
			RegisteredClaims: jwt.RegisteredClaims{Subject: "auth-123"},
		})
		c.Next()
	})
	router.POST("/usuarios/registrar", h.Registrar)

	grabador := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/usuarios/registrar", strings.NewReader(cuerpo))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(grabador, request)
	return grabador
}

func TestRegistrar_NombreVacioOAusenteRespondeNombreObligatorio(t *testing.T) {
	cuerpos := map[string]string{
		"vacío":         `{"nombre": ""}`,
		"solo espacios": `{"nombre": "   "}`,
		"ausente":       `{}`,
	}

	for caso, cuerpo := range cuerpos {
		t.Run(caso, func(t *testing.T) {
			servicio := &usuarioServiceFalso{}
			grabador := ejecutarRegistrar(t, servicio, cuerpo)

			if grabador.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, quería 400", grabador.Code)
			}
			if !strings.Contains(grabador.Body.String(), "El nombre es obligatorio") {
				t.Fatalf("body = %s, quería \"El nombre es obligatorio\"", grabador.Body.String())
			}
			if !servicio.llamado {
				t.Fatal("el service no fue llamado: el handler cortó antes de validar el nombre")
			}
		})
	}
}

func TestRegistrar_JSONInvalidoRespondeSolicitudInvalida(t *testing.T) {
	grabador := ejecutarRegistrar(t, &usuarioServiceFalso{}, `{nombre`)

	if grabador.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, quería 400", grabador.Code)
	}
	if !strings.Contains(grabador.Body.String(), "Solicitud inválida") {
		t.Fatalf("body = %s, quería \"Solicitud inválida\"", grabador.Body.String())
	}
}
