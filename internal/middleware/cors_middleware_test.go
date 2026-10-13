package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func responderConCors(origen string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Cors([]string{" http://localhost:5173 ", ""}))
	router.GET("/recurso", func(c *gin.Context) { c.Status(http.StatusOK) })

	request := httptest.NewRequest(http.MethodGet, "/recurso", nil)
	request.Header.Set("Origin", origen)
	respuesta := httptest.NewRecorder()
	router.ServeHTTP(respuesta, request)
	return respuesta
}

func TestCors_OrigenPermitidoExponeContentDisposition(t *testing.T) {
	respuesta := responderConCors("http://localhost:5173")

	if got := respuesta.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
	if got := respuesta.Header().Get("Access-Control-Expose-Headers"); got != "Content-Disposition" {
		t.Errorf("Access-Control-Expose-Headers = %q", got)
	}
}

func TestCors_OrigenNoPermitidoNoAgregaHeaders(t *testing.T) {
	respuesta := responderConCors("http://otro.com")

	if got := respuesta.Header().Get("Access-Control-Expose-Headers"); got != "" {
		t.Errorf("no debería exponer headers a un origen no permitido, se obtuvo %q", got)
	}
}

func TestCors_PreflightRespondeSinContenido(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Cors([]string{"http://localhost:5173"}))

	request := httptest.NewRequest(http.MethodOptions, "/recurso", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	respuesta := httptest.NewRecorder()
	router.ServeHTTP(respuesta, request)

	if respuesta.Code != http.StatusNoContent {
		t.Errorf("status = %d, se esperaba 204", respuesta.Code)
	}
}
