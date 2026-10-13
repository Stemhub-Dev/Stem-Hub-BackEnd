package service

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

func punteroInt64(valor int64) *int64 { return &valor }

func reporteDePrueba(tipo string, datos []map[string]interface{}) *dto.ReporteRespuesta {
	return &dto.ReporteRespuesta{
		TipoReporte: tipo,
		Proyecto: dto.ReporteProyectoResponse{
			CodigoProyecto: 1,
			Nombre:         "Maqueta Solista",
			Estado:         "En Progreso",
			Tipo:           "EP",
		},
		Datos:           datos,
		FechaGeneracion: time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC),
	}
}

func TestConstruirReportePDF_GeneraUnPDFPorCadaTipo(t *testing.T) {
	filaPorTipo := map[string]map[string]interface{}{
		ReporteActividad:       {"canciones": int64(3), "versiones": int64(7), "comentarios": int64(12), "colaboradores": int64(4)},
		ReporteHistorial:       {"cancion": "Balada", "numero_version": int64(2), "fecha": time.Now(), "formato": "wav", "notas": nil},
		ReporteParticipacion:   {"colaborador": "Ana", "rol": "Productor", "versiones_comentadas": int64(2), "comentarios": int64(5)},
		ReporteEstadoCanciones: {"cancion": "Balada", "ultima_version": int64(2), "total_versiones": int64(2), "comentarios": int64(3), "comentarios_pendientes": int64(1), "comentarios_resueltos": int64(2)},
	}

	for tipo, fila := range filaPorTipo {
		t.Run(tipo, func(t *testing.T) {
			contenido, err := construirReportePDF(reporteDePrueba(tipo, []map[string]interface{}{fila}))
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if !bytes.HasPrefix(contenido, []byte("%PDF-")) {
				t.Fatalf("el contenido no es un PDF")
			}
		})
	}
}

func TestConstruirReportePDF_ActividadSinFilasNoFalla(t *testing.T) {
	if _, err := construirReportePDF(reporteDePrueba(ReporteActividad, nil)); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestConstruirReportePDF_TablaLargaAgregaPaginas(t *testing.T) {
	datos := make([]map[string]interface{}, 0, 80)
	for indice := 0; indice < 80; indice++ {
		datos = append(datos, map[string]interface{}{"colaborador": "Colaborador con un nombre bastante largo para recortar", "rol": "Productor"})
	}

	contenido, err := construirReportePDF(reporteDePrueba(ReporteParticipacion, datos))
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if paginas := bytes.Count(contenido, []byte("/Type /Page\n")); paginas < 2 {
		t.Errorf("se esperaban varias páginas, hubo %d", paginas)
	}
}

func TestConstruirReportePDF_TipoSinColumnasNoArmaTabla(t *testing.T) {
	if _, err := construirReportePDF(reporteDePrueba("desconocido", nil)); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestRecortar(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Arial", "", 9)
	doc := &documentoReporte{pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor("")}

	if got := doc.recortar("Corto", 100); got != "Corto" {
		t.Errorf("recortar texto corto = %q", got)
	}
	if got := doc.recortar("Canción con un nombre larguísimo", 20); !strings.HasSuffix(got, "...") {
		t.Errorf("recortar texto largo = %q, se esperaban puntos suspensivos", got)
	}
	if got := doc.recortar("Canción", 0.1); got != "C" {
		t.Errorf("recortar sin espacio = %q, se esperaba la primera runa", got)
	}
}

func TestDescribirFiltros(t *testing.T) {
	desde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	estado := "Pendiente"
	vacio := "  "

	casos := []struct {
		nombre     string
		tipo       string
		parametros dto.GenerarReporteRequest
		datos      []map[string]interface{}
		esperado   []string
	}{
		{
			nombre:     "actividad sin fechas",
			tipo:       ReporteActividad,
			parametros: dto.GenerarReporteRequest{EstadoComentario: &vacio},
			esperado:   []string{"Desde: sin límite", "Hasta: sin límite", "Las fechas aplican a versiones y comentarios"},
		},
		{
			nombre:     "historial con canción y versión encontradas en las filas",
			tipo:       ReporteHistorial,
			parametros: dto.GenerarReporteRequest{FechaDesde: &desde, CodigoCancion: punteroInt64(5), CodigoVersion: punteroInt64(9)},
			datos: []map[string]interface{}{
				{"codigo_cancion": "no es un número"},
				{"otra": int64(1)},
				{"codigo_cancion": int64(5), "cancion": "Balada", "codigo_version": int64(9), "numero_version": int64(3)},
			},
			esperado: []string{"Desde: 01/01/2026", "Hasta: sin límite", "Canción: Balada", "Versión: v3"},
		},
		{
			nombre:     "historial con códigos que no aparecen en las filas",
			tipo:       ReporteHistorial,
			parametros: dto.GenerarReporteRequest{FechaHasta: &desde, CodigoCancion: punteroInt64(5), CodigoVersion: punteroInt64(9)},
			esperado:   []string{"Desde: sin límite", "Hasta: 01/01/2026", "Canción: #5", "Versión: #9"},
		},
		{
			nombre:     "estado de canciones sin canción elegida y con estado de comentario",
			tipo:       ReporteEstadoCanciones,
			parametros: dto.GenerarReporteRequest{EstadoComentario: &estado},
			esperado:   []string{"Canción: todas", "Estado de comentarios: Pendiente"},
		},
		{
			nombre:   "historial sin filtros",
			tipo:     ReporteHistorial,
			esperado: []string{"Desde: sin límite", "Hasta: sin límite", "Canción: todas", "Versión: todas"},
		},
		{
			nombre:   "tipo sin filtros aplicables",
			tipo:     "desconocido",
			esperado: []string{"Sin filtros: todo el proyecto"},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			reporte := reporteDePrueba(caso.tipo, caso.datos)
			reporte.Parametros = caso.parametros

			if got := describirFiltros(reporte); !reflect.DeepEqual(got, caso.esperado) {
				t.Errorf("describirFiltros() = %q, se esperaba %q", got, caso.esperado)
			}
		})
	}
}

type valorSinFormato struct{ texto string }

func TestValorATexto(t *testing.T) {
	casos := []struct {
		valor    interface{}
		esperado string
	}{
		{nil, sinDato},
		{"   ", sinDato},
		{"Balada", "Balada"},
		{time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), "29/09/2026"},
		{7, "7"},
		{int64(8), "8"},
		{2.5, "2.5"},
		{true, "Sí"},
		{false, "No"},
		{valorSinFormato{"x"}, "{x}"},
	}

	for _, caso := range casos {
		if got := valorATexto(caso.valor); got != caso.esperado {
			t.Errorf("valorATexto(%#v) = %q, se esperaba %q", caso.valor, got, caso.esperado)
		}
	}
}

func TestTituloReporte(t *testing.T) {
	if got := tituloReporte(ReporteEstadoCanciones); got != "Estado de canciones" {
		t.Errorf("tituloReporte conocido = %q", got)
	}
	if got := tituloReporte("otro"); got != "otro" {
		t.Errorf("tituloReporte desconocido = %q", got)
	}
}

func TestCantidadDeFilas(t *testing.T) {
	if got := cantidadDeFilas(1); got != "1 resultado" {
		t.Errorf("cantidadDeFilas(1) = %q", got)
	}
	if got := cantidadDeFilas(3); got != "3 resultados" {
		t.Errorf("cantidadDeFilas(3) = %q", got)
	}
}

func TestSanitizarNombre(t *testing.T) {
	casos := map[string]string{
		"  Maqueta Solista  ":   "maqueta-solista",
		"Canción / Ñandú (v2)":  "cancion-nandu-v2",
		"Estado de canciones":   "estado-de-canciones",
		"!!!":                   "reporte",
		"--Álbum__Único--":      "album-unico",
	}

	for entrada, esperado := range casos {
		if got := sanitizarNombre(entrada); got != esperado {
			t.Errorf("sanitizarNombre(%q) = %q, se esperaba %q", entrada, got, esperado)
		}
	}
}

func TestNombreArchivoReporte(t *testing.T) {
	anterior := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = anterior })

	got := nombreArchivoReporte(reporteDePrueba(ReporteEstadoCanciones, nil))
	if esperado := "maqueta-solista_estado-de-canciones_2026-09-29.pdf"; got != esperado {
		t.Errorf("nombreArchivoReporte() = %q, se esperaba %q", got, esperado)
	}
}
