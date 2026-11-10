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

func filaEstado(cancion string, comentarios, resueltos, pendientes int64) map[string]interface{} {
	return map[string]interface{}{
		"cancion": cancion, "ultima_version": int64(2), "total_versiones": int64(2),
		"comentarios": comentarios, "comentarios_resueltos": resueltos, "comentarios_pendientes": pendientes,
	}
}

func TestConstruirReportePDF_GeneraElEstadoDelProyecto(t *testing.T) {
	contenido, err := construirReportePDF(reporteDePrueba(ReporteEstadoCanciones, []map[string]interface{}{
		filaEstado("Balada", 3, 2, 1),
	}), nil)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !bytes.HasPrefix(contenido, []byte("%PDF-")) {
		t.Fatalf("el contenido no es un PDF")
	}
}

func TestConstruirReportePDF_SinFilasNoFalla(t *testing.T) {
	if _, err := construirReportePDF(reporteDePrueba(ReporteEstadoCanciones, nil), nil); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	filtrado := reporteDePrueba(ReporteEstadoCanciones, nil)
	filtrado.Parametros.CodigoCancion = punteroInt64(5)
	if _, err := construirReportePDF(filtrado, nil); err != nil {
		t.Fatalf("error inesperado con filtro: %v", err)
	}
}

func TestConstruirReportePDF_TablaLargaAgregaPaginas(t *testing.T) {
	datos := make([]map[string]interface{}, 0, 80)
	for indice := 0; indice < 80; indice++ {
		datos = append(datos, filaEstado("Canción con un nombre bastante largo para tener que recortarlo", 1, 1, 0))
	}

	contenido, err := construirReportePDF(reporteDePrueba(ReporteEstadoCanciones, datos), nil)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if paginas := bytes.Count(contenido, []byte("/Type /Page\n")); paginas < 2 {
		t.Errorf("se esperaban varias páginas, hubo %d", paginas)
	}
}

func TestConstruirReportePDF_TipoDesconocidoNoFalla(t *testing.T) {
	if _, err := construirReportePDF(reporteDePrueba("desconocido", nil), nil); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestTotalesEstado(t *testing.T) {
	totales := totalesEstado([]map[string]interface{}{
		filaEstado("Balada", 4, 3, 1),
		filaEstado("Rock", 2, 2, 0),
		filaEstado("Intro", 0, 0, 0),
	})
	esperado := totalesProyecto{canciones: 3, versiones: 6, comentarios: 6, resueltos: 5, conPendientes: 1}
	if totales != esperado {
		t.Errorf("totalesEstado() = %+v, se esperaba %+v", totales, esperado)
	}
	// Redondea hacia abajo, igual que el frontend.
	if got := totales.porcentajeResueltos(); got != "83%" {
		t.Errorf("porcentajeResueltos() = %q", got)
	}
	if got := totalesEstado(nil).porcentajeResueltos(); got != sinDato {
		t.Errorf("porcentajeResueltos() sin comentarios = %q", got)
	}
}

func TestEstadoCancion(t *testing.T) {
	casos := []struct {
		nombre string
		fila   map[string]interface{}
		quiero string
	}{
		{"sin versiones", map[string]interface{}{"ultima_version": int64(0)}, "Sin versiones"},
		{"un pendiente", filaEstado("A", 3, 2, 1), "1 pendiente"},
		{"varios pendientes", filaEstado("A", 3, 0, 3), "3 pendientes"},
		{"sin comentarios", filaEstado("A", 0, 0, 0), "Sin revisar"},
		{"todo resuelto", filaEstado("A", 2, 2, 0), "Al día"},
	}
	for _, caso := range casos {
		if got := estadoCancion(caso.fila).texto; got != caso.quiero {
			t.Errorf("%s: estadoCancion() = %q, quería %q", caso.nombre, got, caso.quiero)
		}
	}
}

// La misma canción tiene que tener siempre la misma portada.
func TestIndicePortadaEsEstable(t *testing.T) {
	if indicePortada("Balada") != indicePortada("  balada ") {
		t.Error("el índice depende de mayúsculas o espacios")
	}
	vistos := map[int]bool{}
	for _, nombre := range []string{"Balada", "Rock", "Intro", "Outro", "Noches de Sur", "Marea Alta", "Casa Vacía"} {
		indice := indicePortada(nombre)
		if indice < 0 || indice >= len(disenosPortada) {
			t.Fatalf("índice fuera de rango: %d", indice)
		}
		vistos[indice] = true
	}
	if len(vistos) < 2 {
		t.Error("todas las canciones recibieron la misma portada")
	}
}

func TestColorHex(t *testing.T) {
	if got := colorHex("#6f62a0"); got != (colorRGB{111, 98, 160}) {
		t.Errorf("colorHex = %+v", got)
	}
	if got := colorHex("nada"); got != (colorRGB{}) {
		t.Errorf("colorHex inválido = %+v", got)
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
	estado := "Pendiente"
	vacio := "  "

	casos := []struct {
		nombre     string
		parametros dto.GenerarReporteRequest
		datos      []map[string]interface{}
		esperado   []string
	}{
		{nombre: "sin filtros no aclara nada", esperado: []string{}},
		{
			nombre:     "estado de comentario en blanco no se lista",
			parametros: dto.GenerarReporteRequest{EstadoComentario: &vacio},
			esperado:   []string{},
		},
		{
			nombre:     "canción encontrada en las filas y estado de comentario",
			parametros: dto.GenerarReporteRequest{CodigoCancion: punteroInt64(5), EstadoComentario: &estado},
			datos: []map[string]interface{}{
				{"codigo_cancion": "no es un número"},
				{"otra": int64(1)},
				{"codigo_cancion": int64(5), "cancion": "Balada"},
			},
			esperado: []string{"Canción: Balada", "Comentarios: Pendiente"},
		},
		{
			nombre:     "canción que no aparece en las filas",
			parametros: dto.GenerarReporteRequest{CodigoCancion: punteroInt64(5)},
			esperado:   []string{"Canción: #5"},
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			reporte := reporteDePrueba(ReporteEstadoCanciones, caso.datos)
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
	if got := tituloReporte(ReporteEstadoCanciones); got != "Estado del proyecto" {
		t.Errorf("tituloReporte conocido = %q", got)
	}
	if got := tituloReporte("otro"); got != "otro" {
		t.Errorf("tituloReporte desconocido = %q", got)
	}
}

func TestPlural(t *testing.T) {
	if got := plural(1, "canción", "canciones"); got != "1 canción" {
		t.Errorf("plural(1) = %q", got)
	}
	if got := plural(3, "canción", "canciones"); got != "3 canciones" {
		t.Errorf("plural(3) = %q", got)
	}
}

func TestSanitizarNombre(t *testing.T) {
	casos := map[string]string{
		"  Maqueta Solista  ":  "maqueta-solista",
		"Canción / Ñandú (v2)": "cancion-nandu-v2",
		"Estado de canciones":  "estado-de-canciones",
		"!!!":                  "reporte",
		"--Álbum__Único--":     "album-unico",
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
	if esperado := "maqueta-solista_estado-del-proyecto_2026-09-29.pdf"; got != esperado {
		t.Errorf("nombreArchivoReporte() = %q, se esperaba %q", got, esperado)
	}
}

// El PDF muestra las versiones igual que el resto de la app ("v3").
func TestVersionATexto(t *testing.T) {
	casos := []struct {
		valor  interface{}
		quiero string
	}{
		{int64(3), "v3"},
		{int64(0), sinDato},
		{nil, sinDato},
	}

	for _, caso := range casos {
		if got := versionATexto(caso.valor); got != caso.quiero {
			t.Errorf("versionATexto(%#v) = %q, quería %q", caso.valor, got, caso.quiero)
		}
	}
}
