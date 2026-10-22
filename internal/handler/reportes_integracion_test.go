package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

// Tests de integración de HU-REP-B01 (listado de reportes), HU-REP-B02
// (generar) y HU-REP-B03 (exportar a PDF) sobre el escenario de versiones:
// el productor es el propietario y el músico, no. Que falle el armado del
// PDF (HTTP 500) no se puede provocar con datos reales; el mensaje lo cubre
// el default de ReporteHandler.responderError.

func prepararEscenarioReportes(t *testing.T) *escenarioVersiones {
	t.Helper()
	e := prepararEscenarioVersiones(t)

	h := handler.NewReporteHandler(service.NewReporteService(
		repository.NewReporteRepository(e.db),
		repository.NewProyectoRepository(e.db),
		repository.NewIntegranteRepository(e.db),
	))
	e.router.GET("/report/reportes", h.Listar)
	e.router.POST("/report/reportes/:tipo/generar", h.Generar)
	e.router.POST("/report/reportes/:tipo/exportar-pdf", h.ExportarPDF)
	return e
}

func (e *escenarioVersiones) pedirReporte(t *testing.T, usuario int64, metodo, ruta string, cuerpo any) *httptest.ResponseRecorder {
	t.Helper()
	datos, _ := json.Marshal(cuerpo)
	req := httptest.NewRequest(metodo, ruta, bytes.NewReader(datos))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Usuario", fmt.Sprint(usuario))
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

func errorDe(rec *httptest.ResponseRecorder) string {
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	mensaje, _ := body["error"].(string)
	return mensaje
}

// HU-REP-B01

func TestReportes_ListarLosCuatroDelProyecto(t *testing.T) {
	e := prepararEscenarioReportes(t)

	rec := e.pedirReporte(t, e.productor, http.MethodGet, fmt.Sprintf("/report/reportes?proyectoId=%d", e.proyecto), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	var reportes []struct {
		ID      string `json:"id_reporte"`
		Tipo    string `json:"tipo_reporte"`
		Funcion string `json:"funcion_ejecucion_reporte"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reportes); err != nil {
		t.Fatal(err)
	}
	esperados := map[string]string{
		"REP-001": "actividad-proyecto",
		"REP-002": "historial-versiones",
		"REP-003": "participacion-colaboradores",
		"REP-004": "estado-canciones",
	}
	if len(reportes) != len(esperados) {
		t.Fatalf("reportes = %+v", reportes)
	}
	for _, r := range reportes {
		if esperados[r.ID] != r.Tipo || r.Funcion == "" {
			t.Fatalf("reporte inesperado: %+v", r)
		}
	}
}

func TestReportes_ListarErrores(t *testing.T) {
	e := prepararEscenarioReportes(t)

	casos := []struct {
		nombre  string
		usuario int64
		query   string
		status  int
		mensaje string
	}{
		{"sin proyectoId", e.productor, "", http.StatusBadRequest, "proyectoId es obligatorio y debe ser válido"},
		{"proyectoId inválido", e.productor, "?proyectoId=abc", http.StatusBadRequest,
			"proyectoId es obligatorio y debe ser válido"},
		{"no es el propietario", e.musico, fmt.Sprintf("?proyectoId=%d", e.proyecto), http.StatusForbidden,
			"No tenés permiso para consultar reportes de este proyecto"},
		{"proyecto inexistente", e.productor, "?proyectoId=999999999", http.StatusNotFound, "El proyecto no existe"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			rec := e.pedirReporte(t, caso.usuario, http.MethodGet, "/report/reportes"+caso.query, nil)
			if rec.Code != caso.status || errorDe(rec) != caso.mensaje {
				t.Fatalf("status = %d, error = %q; quería %d %q", rec.Code, errorDe(rec), caso.status, caso.mensaje)
			}
		})
	}
}

// HU-REP-B02

func TestReportes_GenerarHistorialDeVersiones(t *testing.T) {
	e := prepararEscenarioReportes(t)

	rec := e.pedirReporte(t, e.productor, http.MethodPost, "/report/reportes/historial-versiones/generar",
		map[string]any{"codigo_proyecto": e.proyecto})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	var reporte struct {
		ID         string           `json:"id_reporte"`
		Tipo       string           `json:"tipo_reporte"`
		Parametros map[string]any   `json:"parametros"`
		Proyecto   map[string]any   `json:"proyecto"`
		Datos      []map[string]any `json:"datos"`
		Fecha      string           `json:"fecha_generacion"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reporte); err != nil {
		t.Fatal(err)
	}
	if reporte.ID != "REP-002" || reporte.Tipo != "historial-versiones" || reporte.Fecha == "" {
		t.Fatalf("reporte = %+v", reporte)
	}
	if reporte.Parametros["codigo_proyecto"] != float64(e.proyecto) || reporte.Proyecto == nil {
		t.Fatalf("parámetros o proyecto = %v %v", reporte.Parametros, reporte.Proyecto)
	}
	if len(reporte.Datos) != 3 {
		t.Fatalf("filas = %d, quería una por versión (3)", len(reporte.Datos))
	}
}

func TestReportes_GenerarSinDatosResponde204(t *testing.T) {
	e := prepararEscenarioReportes(t)

	// Todas las versiones del escenario son anteriores a este rango.
	rec := e.pedirReporte(t, e.productor, http.MethodPost, "/report/reportes/historial-versiones/generar",
		map[string]any{
			"codigo_proyecto": e.proyecto,
			"fecha_desde":     "2099-01-01T00:00:00Z",
			"fecha_hasta":     "2099-12-31T00:00:00Z",
		})

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestReportes_GenerarErrores(t *testing.T) {
	e := prepararEscenarioReportes(t)

	casos := []struct {
		nombre  string
		usuario int64
		tipo    string
		cuerpo  map[string]any
		status  int
		mensaje string
	}{
		{"sin codigo_proyecto", e.productor, "historial-versiones", map[string]any{}, http.StatusBadRequest,
			"Los parámetros del reporte son inválidos"},
		{"fechas invertidas", e.productor, "historial-versiones", map[string]any{
			"codigo_proyecto": e.proyecto,
			"fecha_desde":     "2026-10-20T00:00:00Z",
			"fecha_hasta":     "2026-10-01T00:00:00Z",
		}, http.StatusBadRequest, "Los filtros del reporte son inválidos"},
		{"tipo inexistente", e.productor, "ventas", map[string]any{"codigo_proyecto": e.proyecto},
			http.StatusNotFound, "El reporte no existe"},
		{"no es el propietario", e.musico, "historial-versiones", map[string]any{"codigo_proyecto": e.proyecto},
			http.StatusForbidden, "No tenés permiso para consultar reportes de este proyecto"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			rec := e.pedirReporte(t, caso.usuario, http.MethodPost, "/report/reportes/"+caso.tipo+"/generar", caso.cuerpo)
			if rec.Code != caso.status || errorDe(rec) != caso.mensaje {
				t.Fatalf("status = %d, error = %q; quería %d %q", rec.Code, errorDe(rec), caso.status, caso.mensaje)
			}
		})
	}
}

// HU-REP-B03

func TestReportes_ExportarPDF(t *testing.T) {
	e := prepararEscenarioReportes(t)

	rec := e.pedirReporte(t, e.productor, http.MethodPost, "/report/reportes/historial-versiones/exportar-pdf",
		map[string]any{"codigo_proyecto": e.proyecto})

	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("status = %d, content-type = %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	disposicion := rec.Header().Get("Content-Disposition")
	if !strings.HasPrefix(disposicion, "attachment;") || !strings.Contains(disposicion, ".pdf") {
		t.Fatalf("Content-Disposition = %q", disposicion)
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF")) {
		t.Fatal("el cuerpo no es un PDF")
	}

	rec = e.pedirReporte(t, e.productor, http.MethodPost, "/report/reportes/historial-versiones/exportar-pdf",
		map[string]any{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("parámetros inválidos: status = %d", rec.Code)
	}
}
