package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type ayudaServiceFalso struct {
	preguntas  []dto.PreguntaFrecuenteResponse
	err        error
	rutaManual string
	errManual  error
}

func (s *ayudaServiceFalso) ObtenerRutaManual() (string, error) {
	return s.rutaManual, s.errManual
}

func (s *ayudaServiceFalso) ListarPreguntasFrecuentes() ([]dto.PreguntaFrecuenteResponse, error) {
	return s.preguntas, s.err
}

func routerDeAyuda(servicio *ayudaServiceFalso) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ayuda := NewAyudaHandler(servicio)
	router.GET("/ayuda/preguntas-frecuentes", ayuda.ListarPreguntasFrecuentes)
	router.GET("/ayuda/manual", ayuda.DescargarManual)
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

func TestDescargarManual_DevuelveElPDFComoAdjunto(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "manual.pdf")
	if err := os.WriteFile(ruta, []byte("%PDF-1.7 contenido"), 0o600); err != nil {
		t.Fatal(err)
	}

	respuesta := pedirAyuda(routerDeAyuda(&ayudaServiceFalso{rutaManual: ruta}), "/ayuda/manual")

	if respuesta.Code != http.StatusOK {
		t.Fatalf("status = %d", respuesta.Code)
	}
	if tipo := respuesta.Header().Get("Content-Type"); tipo != "application/pdf" {
		t.Errorf("Content-Type = %q", tipo)
	}
	if adjunto := respuesta.Header().Get("Content-Disposition"); !strings.Contains(adjunto, "attachment") ||
		!strings.Contains(adjunto, "manual-usuario-stemhub.pdf") {
		t.Errorf("Content-Disposition = %q", adjunto)
	}
	if respuesta.Body.String() != "%PDF-1.7 contenido" {
		t.Errorf("body = %q", respuesta.Body.String())
	}
}

func TestDescargarManual_SinManualDevuelve404(t *testing.T) {
	respuesta := pedirAyuda(routerDeAyuda(&ayudaServiceFalso{
		errManual: service.ErrManualUsuarioNoDisponible,
	}), "/ayuda/manual")

	if respuesta.Code != http.StatusNotFound {
		t.Fatalf("status = %d", respuesta.Code)
	}
}

func TestDescargarManual_ErrorDevuelve500(t *testing.T) {
	respuesta := pedirAyuda(routerDeAyuda(&ayudaServiceFalso{
		errManual: errors.New("permiso denegado"),
	}), "/ayuda/manual")

	if respuesta.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", respuesta.Code)
	}
}
