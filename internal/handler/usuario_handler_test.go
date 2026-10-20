package handler

import (
	"encoding/json"
	"fmt"
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

// HU-ACA-B01: POST /usuarios/registrar.

// Valida el nombre igual que el service real (el handler no debe cortar
// antes con su propio mensaje) y devuelve lo que se le configure.
type usuarioServiceFalso struct {
	llamado bool
	creado  bool
	err     error
}

func (s *usuarioServiceFalso) ObtenerUsuarioActivoPorIDAutenticacion(string) (*model.Usuario, error) {
	return nil, nil
}

func (s *usuarioServiceFalso) RegistrarUsuario(
	_ string,
	email string,
	nombre string,
	descripcion *string,
) (*model.Usuario, *model.Integrante, bool, error) {
	s.llamado = true
	if strings.TrimSpace(nombre) == "" {
		return nil, nil, false, service.ErrNombreIntegranteObligatorio
	}
	if s.err != nil {
		return nil, nil, false, s.err
	}
	return &model.Usuario{CodigoUsuario: 7, Email: email},
		&model.Integrante{CodIntegrante: 3, NombreIntegrante: strings.TrimSpace(nombre), DescripcionIntegrante: descripcion},
		s.creado,
		nil
}

var claimsAna = &middleware.SupabaseClaims{
	Email:            "ana@example.com",
	RegisteredClaims: jwt.RegisteredClaims{Subject: "auth-123"},
}

func ejecutarRegistrarCon(
	t *testing.T,
	servicio *usuarioServiceFalso,
	claims *middleware.SupabaseClaims,
	cuerpo string,
) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h := NewUsuarioHandler(servicio, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if claims != nil {
			c.Set(middleware.ClaimsContextKey, claims)
		}
		c.Next()
	})
	router.POST("/usuarios/registrar", h.Registrar)

	grabador := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/usuarios/registrar", strings.NewReader(cuerpo))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(grabador, request)
	return grabador
}

func ejecutarRegistrar(t *testing.T, servicio *usuarioServiceFalso, cuerpo string) *httptest.ResponseRecorder {
	t.Helper()
	return ejecutarRegistrarCon(t, servicio, claimsAna, cuerpo)
}

func esperarError(t *testing.T, grabador *httptest.ResponseRecorder, status int, mensaje string) {
	t.Helper()

	if grabador.Code != status {
		t.Fatalf("status = %d, quería %d (body %s)", grabador.Code, status, grabador.Body.String())
	}
	if !strings.Contains(grabador.Body.String(), mensaje) {
		t.Fatalf("body = %s, quería %q", grabador.Body.String(), mensaje)
	}
}

func TestRegistrar_PrimeraVezCreaYRespondeCreated(t *testing.T) {
	grabador := ejecutarRegistrar(t, &usuarioServiceFalso{creado: true}, `{"nombre": "Ana", "descripcion": "Bajista"}`)

	if grabador.Code != http.StatusCreated {
		t.Fatalf("status = %d, quería 201", grabador.Code)
	}

	var respuesta map[string]any
	if err := json.Unmarshal(grabador.Body.Bytes(), &respuesta); err != nil {
		t.Fatal(err)
	}
	for campo, quiero := range map[string]any{
		"codigoUsuario":    float64(7),
		"codigoIntegrante": float64(3),
		"email":            "ana@example.com",
		"nombre":           "Ana",
		"descripcion":      "Bajista",
	} {
		if fmt.Sprint(respuesta[campo]) != fmt.Sprint(quiero) {
			t.Errorf("%s = %v, quería %v", campo, respuesta[campo], quiero)
		}
	}
}

func TestRegistrar_YaRegistradoRespondeOK(t *testing.T) {
	grabador := ejecutarRegistrar(t, &usuarioServiceFalso{creado: false}, `{"nombre": "Ana"}`)

	if grabador.Code != http.StatusOK {
		t.Fatalf("status = %d, quería 200", grabador.Code)
	}
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

			esperarError(t, grabador, http.StatusBadRequest, "El nombre es obligatorio")
			if !servicio.llamado {
				t.Fatal("el service no fue llamado: el handler cortó antes de validar el nombre")
			}
		})
	}
}

func TestRegistrar_NombreLargo(t *testing.T) {
	grabador := ejecutarRegistrar(t, &usuarioServiceFalso{err: service.ErrNombreIntegranteLargo}, `{"nombre": "Ana"}`)

	esperarError(t, grabador, http.StatusBadRequest, "El nombre no puede superar los 50 caracteres")
}

func TestRegistrar_TokenSinEmail(t *testing.T) {
	claims := &middleware.SupabaseClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "auth-123"}}
	servicio := &usuarioServiceFalso{}

	grabador := ejecutarRegistrarCon(t, servicio, claims, `{"nombre": "Ana"}`)

	esperarError(t, grabador, http.StatusBadRequest, "el token no contiene email")
	if servicio.llamado {
		t.Fatal("no debería registrar sin email")
	}
}

func TestRegistrar_UsuarioDadoDeBaja(t *testing.T) {
	grabador := ejecutarRegistrar(t, &usuarioServiceFalso{err: service.ErrUsuarioInactivo}, `{"nombre": "Ana"}`)

	esperarError(t, grabador, http.StatusForbidden, "usuario inactivo")
}

func TestRegistrar_SinIdentidadRespondeUnauthorized(t *testing.T) {
	grabador := ejecutarRegistrarCon(t, &usuarioServiceFalso{}, nil, `{"nombre": "Ana"}`)

	if grabador.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, quería 401", grabador.Code)
	}
}

func TestRegistrar_JSONInvalidoRespondeSolicitudInvalida(t *testing.T) {
	grabador := ejecutarRegistrar(t, &usuarioServiceFalso{}, `{nombre`)

	esperarError(t, grabador, http.StatusBadRequest, "Solicitud inválida")
}
