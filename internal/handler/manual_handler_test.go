package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type mockFileStorage struct {
	descargarArchivoFunc func(ctx context.Context, bucket, objectName string) ([]byte, error)
}

func (m *mockFileStorage) DescargarArchivo(ctx context.Context, bucket, objectName string) ([]byte, error) {
	if m.descargarArchivoFunc != nil {
		return m.descargarArchivoFunc(ctx, bucket, objectName)
	}
	return nil, errors.New("archivo no encontrado")
}

func TestDescargarManual_Exitoso(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pdfEsperado := []byte("%PDF-1.4 contenido simulado del manual")
	mockStorage := &mockFileStorage{
		descargarArchivoFunc: func(ctx context.Context, bucket, objectName string) ([]byte, error) {
			if bucket != "documentos" || objectName != "manual-usuario.pdf" {
				t.Fatalf("bucket u objectName inesperado: %s / %s", bucket, objectName)
			}
			return pdfEsperado, nil
		},
	}

	manualService := service.NewManualService(mockStorage, "documentos", "manual-usuario.pdf")
	manualHandler := handler.NewManualHandler(manualService)

	router := gin.New()
	router.GET("/manual-usuario/pdf", manualHandler.DescargarManual)

	req := httptest.NewRequest(http.MethodGet, "/manual-usuario/pdf", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("esperado status 200, obtenido %d", resp.Code)
	}

	if cd := resp.Header().Get("Content-Disposition"); cd != `attachment; filename="manual-usuario.pdf"` {
		t.Fatalf("Content-Disposition esperado 'attachment; filename=\"manual-usuario.pdf\"', obtenido '%s'", cd)
	}

	if ct := resp.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("Content-Type esperado 'application/pdf', obtenido '%s'", ct)
	}

	if resp.Body.String() != string(pdfEsperado) {
		t.Fatalf("cuerpo de respuesta no coincide con los bytes esperados")
	}
}

func TestDescargarManual_NoEncontrado(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockStorage := &mockFileStorage{
		descargarArchivoFunc: func(ctx context.Context, bucket, objectName string) ([]byte, error) {
			return nil, errors.New("no such key")
		},
	}

	manualService := service.NewManualService(mockStorage, "documentos", "manual-usuario.pdf")
	manualHandler := handler.NewManualHandler(manualService)

	router := gin.New()
	router.GET("/manual-usuario/pdf", manualHandler.DescargarManual)

	req := httptest.NewRequest(http.MethodGet, "/manual-usuario/pdf", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("esperado status 404, obtenido %d", resp.Code)
	}

	cuerpoEsperado := `{"error":"manual de usuario no encontrado"}`
	if resp.Body.String() != cuerpoEsperado {
		t.Fatalf("cuerpo esperado '%s', obtenido '%s'", cuerpoEsperado, resp.Body.String())
	}
}
