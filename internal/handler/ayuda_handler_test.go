package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

type ayudaServiceFalso struct {
	preguntas []dto.PreguntaFrecuenteResponse
	err       error
}

func (s *ayudaServiceFalso) ListarPreguntasFrecuentes() ([]dto.PreguntaFrecuenteResponse, error) {
	return s.preguntas, s.err
}

func routerDeAyuda(servicio *ayudaServiceFalso) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ayuda := NewAyudaHandler(servicio)
	router.GET("/ayuda/preguntas-frecuentes", ayuda.ListarPreguntasFrecuentes)
	return router
}

func pedirAyuda(router *gin.Engine, url string) *httptest.ResponseRecorder {
	respuesta := httptest.NewRecorder()
	router.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, url, nil))
	return respuesta
}

func TestListarPreguntasFrecuentesHandler_DevuelveLosCamposDeLaHU(t *testing.T) {
	respuesta := pedirAyuda(routerDeAyuda(&ayudaServiceFalso{
		preguntas: []dto.PreguntaFrecuenteResponse{{
			ID:                         1,
			FuncionalidadPrincipal:     "Registro y acceso",
			PreguntaFrecuente:          "¿Cómo creo una cuenta?",
			RespuestaPreguntaFrecuente: "Con tu email.",
		}},
	}), "/ayuda/preguntas-frecuentes")

	if respuesta.Code != http.StatusOK {
		t.Fatalf("status = %d", respuesta.Code)
	}
	for _, campo := range []string{`"id":1`, `"funcionalidadPrincipal"`, `"preguntaFrecuente"`, `"respuestaPreguntaFrecuente"`} {
		if !strings.Contains(respuesta.Body.String(), campo) {
			t.Errorf("falta %s en %s", campo, respuesta.Body.String())
		}
	}
}

func TestListarPreguntasFrecuentesHandler_SinPreguntasDevuelveArrayVacio(t *testing.T) {
	respuesta := pedirAyuda(routerDeAyuda(&ayudaServiceFalso{
		preguntas: []dto.PreguntaFrecuenteResponse{},
	}), "/ayuda/preguntas-frecuentes")

	if respuesta.Code != http.StatusOK || strings.TrimSpace(respuesta.Body.String()) != "[]" {
		t.Fatalf("status = %d, body = %s", respuesta.Code, respuesta.Body.String())
	}
}

func TestListarPreguntasFrecuentesHandler_ErrorDevuelve500(t *testing.T) {
	respuesta := pedirAyuda(routerDeAyuda(&ayudaServiceFalso{
		err: errors.New("fallo de base"),
	}), "/ayuda/preguntas-frecuentes")

	if respuesta.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", respuesta.Code)
	}
}
