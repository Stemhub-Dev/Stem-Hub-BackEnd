package service

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

// Colores tomados del design system del frontend
// (stemhub-frontend/src/styles/tokens.css) para que el PDF se lea como parte
// de la aplicación y no como un volcado de datos.
type colorRGB struct{ r, g, b int }

var (
	colorLavender500 = colorRGB{155, 142, 196} // #9b8ec4
	colorLavender700 = colorRGB{111, 98, 160}  // #6f62a0
	colorCream100    = colorRGB{247, 245, 242} // #f7f5f2
	colorLinea       = colorRGB{230, 225, 218} // #e6e1da
	colorInk900      = colorRGB{44, 42, 53}    // #2c2a35
	colorInk500      = colorRGB{110, 107, 122} // #6e6b7a
	colorBlanco      = colorRGB{255, 255, 255}
)

const (
	margenPDF      = 15.0
	anchoPaginaA4  = 210.0
	altoPaginaA4   = 297.0
	anchoUtilPDF   = anchoPaginaA4 - (margenPDF * 2)
	altoFilaTabla  = 7.5
	limiteInferior = altoPaginaA4 - 20
	sinDato        = "-"
	formatoFecha   = "02/01/2006"
	sinLimite      = "sin límite"
)

// Títulos legibles: el slug del tipo ("estado-canciones") sirve para la URL,
// no para mostrárselo a una persona.
var titulosReporte = map[string]string{
	ReporteActividad:       "Actividad del proyecto",
	ReporteHistorial:       "Historial de versiones",
	ReporteParticipacion:   "Participación de colaboradores",
	ReporteEstadoCanciones: "Estado de canciones",
}

// columnaPDF describe una columna de la tabla. El ancho es la proporción del
// ancho útil de la hoja (las proporciones de cada tipo suman 1).
type columnaPDF struct {
	clave    string
	etiqueta string
	ancho    float64
	alinear  string
}

// Mismas claves y etiquetas que la vista previa del frontend
// (stemhub-frontend/src/features/reports/columnasReporte.ts): el PDF y la
// pantalla tienen que mostrar lo mismo.
var columnasPorTipo = map[string][]columnaPDF{
	ReporteHistorial: {
		{clave: "cancion", etiqueta: "Canción", ancho: 0.32, alinear: "L"},
		{clave: "numero_version", etiqueta: "Versión", ancho: 0.12, alinear: "C"},
		{clave: "fecha", etiqueta: "Fecha", ancho: 0.17, alinear: "L"},
		{clave: "formato", etiqueta: "Formato", ancho: 0.13, alinear: "C"},
		{clave: "notas", etiqueta: "Notas", ancho: 0.26, alinear: "L"},
	},
	ReporteParticipacion: {
		{clave: "colaborador", etiqueta: "Colaborador", ancho: 0.34, alinear: "L"},
		{clave: "rol", etiqueta: "Rol", ancho: 0.26, alinear: "L"},
		{clave: "versiones_comentadas", etiqueta: "Versiones comentadas", ancho: 0.22, alinear: "C"},
		{clave: "comentarios", etiqueta: "Comentarios", ancho: 0.18, alinear: "C"},
	},
	ReporteEstadoCanciones: {
		{clave: "cancion", etiqueta: "Canción", ancho: 0.31, alinear: "L"},
		{clave: "ultima_version", etiqueta: "Última versión", ancho: 0.15, alinear: "C"},
		{clave: "total_versiones", etiqueta: "Versiones", ancho: 0.13, alinear: "C"},
		{clave: "comentarios", etiqueta: "Comentarios", ancho: 0.15, alinear: "C"},
		{clave: "comentarios_pendientes", etiqueta: "Pendientes", ancho: 0.13, alinear: "C"},
		{clave: "comentarios_resueltos", etiqueta: "Resueltos", ancho: 0.13, alinear: "C"},
	},
}

// filtrosAplicables refleja qué filtros usa de verdad la consulta de cada
// tipo en reporte_repository.go. El bloque "Filtros aplicados" solo lista
// esos: mostrar "Canción: X" en un reporte que la ignora haría creer que los
// números están filtrados cuando no lo están. Mantener en sync con
// FILTROS_POR_TIPO del frontend (features/reports/columnasReporte.ts).
type filtrosAplicables struct{ fechas, cancion, version bool }

var filtrosPorTipo = map[string]filtrosAplicables{
	ReporteActividad:       {fechas: true},
	ReporteHistorial:       {fechas: true, cancion: true, version: true},
	ReporteParticipacion:   {fechas: true},
	ReporteEstadoCanciones: {cancion: true},
}

// actividad-proyecto es una única fila agregada: se muestra como tarjetas de
// métrica, no como tabla de una sola línea.
var metricasActividad = []columnaPDF{
	{clave: "canciones", etiqueta: "Canciones"},
	{clave: "versiones", etiqueta: "Versiones"},
	{clave: "comentarios", etiqueta: "Comentarios"},
	{clave: "colaboradores", etiqueta: "Colaboradores"},
}

// documentoReporte envuelve el fpdf con el traductor de codificación: las
// fuentes core (Arial) son cp1252, sin traducir "Canción" sale con basura.
type documentoReporte struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

func (d *documentoReporte) fondo(color colorRGB) { d.pdf.SetFillColor(color.r, color.g, color.b) }
func (d *documentoReporte) texto(color colorRGB) { d.pdf.SetTextColor(color.r, color.g, color.b) }
func (d *documentoReporte) trazo(color colorRGB) { d.pdf.SetDrawColor(color.r, color.g, color.b) }
func (d *documentoReporte) ancho(texto string) float64 {
	return d.pdf.GetStringWidth(d.tr(texto))
}

// recortar deja el texto que entra en el ancho disponible, con puntos
// suspensivos. Corta por runas: cortar por bytes partiría un carácter UTF-8.
func (d *documentoReporte) recortar(texto string, disponible float64) string {
	if d.ancho(texto) <= disponible {
		return texto
	}
	runas := []rune(texto)
	for len(runas) > 1 {
		runas = runas[:len(runas)-1]
		if d.ancho(string(runas)+"...") <= disponible {
			return strings.TrimRight(string(runas), " ") + "..."
		}
	}
	return string(runas)
}

func construirReportePDF(reporte *dto.ReporteRespuesta) ([]byte, error) {
	titulo := tituloReporte(reporte.TipoReporte)

	pdf := fpdf.New("P", "mm", "A4", "")
	doc := &documentoReporte{pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor("")}

	pdf.SetTitle(titulo+" - "+reporte.Proyecto.Nombre, true)
	pdf.SetAuthor("StemHub", true)
	pdf.SetMargins(margenPDF, margenPDF, margenPDF)
	// El salto de página lo maneja la tabla a mano, para no cortar una fila
	// al medio ni perder el encabezado de columnas en la página siguiente.
	pdf.SetAutoPageBreak(false, 0)
	pdf.AliasNbPages("")
	pdf.SetFooterFunc(func() { doc.pie(titulo) })

	pdf.AddPage()
	doc.encabezado(titulo, reporte)
	doc.bloqueFiltros(reporte)

	if reporte.TipoReporte == ReporteActividad {
		doc.tarjetasMetricas(reporte)
	} else {
		doc.tabla(reporte)
	}

	var contenido bytes.Buffer
	if err := pdf.Output(&contenido); err != nil {
		return nil, err
	}
	return contenido.Bytes(), nil
}

// encabezado: barra de color de marca de lado a lado + datos del proyecto.
func (d *documentoReporte) encabezado(titulo string, reporte *dto.ReporteRespuesta) {
	d.fondo(colorLavender500)
	d.pdf.Rect(0, 0, anchoPaginaA4, 28, "F")

	d.texto(colorBlanco)
	d.pdf.SetFont("Arial", "B", 9)
	d.pdf.SetXY(margenPDF, 7)
	d.pdf.CellFormat(anchoUtilPDF, 5, d.tr("STEMHUB"), "", 1, "L", false, 0, "")

	d.pdf.SetFont("Arial", "B", 18)
	d.pdf.SetX(margenPDF)
	d.pdf.CellFormat(anchoUtilPDF, 10, d.tr(titulo), "", 1, "L", false, 0, "")

	d.pdf.SetY(36)
	d.texto(colorInk900)
	d.pdf.SetFont("Arial", "B", 13)
	d.pdf.CellFormat(anchoUtilPDF, 7, d.tr(reporte.Proyecto.Nombre), "", 1, "L", false, 0, "")

	d.texto(colorInk500)
	d.pdf.SetFont("Arial", "", 9)
	detalle := fmt.Sprintf(
		"%s  ·  %s  ·  Generado el %s",
		reporte.Proyecto.Tipo,
		reporte.Proyecto.Estado,
		reporte.FechaGeneracion.Local().Format(formatoFecha+" 15:04 (UTC-07:00)"),
	)
	d.pdf.CellFormat(anchoUtilPDF, 5, d.tr(detalle), "", 1, "L", false, 0, "")
	d.pdf.Ln(3)
}

// bloqueFiltros muestra los filtros aplicados en lenguaje humano: sin esto el
// PDF no dice a qué recorte de datos corresponde lo que se está leyendo.
func (d *documentoReporte) bloqueFiltros(reporte *dto.ReporteRespuesta) {
	filtros := describirFiltros(reporte)
	y := d.pdf.GetY()
	alto := 16.0

	d.fondo(colorCream100)
	d.trazo(colorLinea)
	d.pdf.SetLineWidth(0.2)
	d.pdf.RoundedRect(margenPDF, y, anchoUtilPDF, alto, 2.5, "1234", "FD")

	d.texto(colorInk500)
	d.pdf.SetFont("Arial", "B", 7.5)
	d.pdf.SetXY(margenPDF+4, y+3)
	d.pdf.CellFormat(anchoUtilPDF-8, 4, d.tr("FILTROS APLICADOS"), "", 1, "L", false, 0, "")

	d.texto(colorInk900)
	d.pdf.SetFont("Arial", "", 9)
	d.pdf.SetX(margenPDF + 4)
	d.pdf.CellFormat(anchoUtilPDF-8, 5, d.tr(d.recortar(strings.Join(filtros, "   ·   "), anchoUtilPDF-8)), "", 1, "L", false, 0, "")

	d.pdf.SetY(y + alto + 6)
}

func (d *documentoReporte) tarjetasMetricas(reporte *dto.ReporteRespuesta) {
	var fila map[string]interface{}
	if len(reporte.Datos) > 0 {
		fila = reporte.Datos[0]
	}

	separacion := 4.0
	anchoTarjeta := (anchoUtilPDF - separacion*float64(len(metricasActividad)-1)) / float64(len(metricasActividad))
	altoTarjeta := 26.0
	y := d.pdf.GetY()

	for indice, metrica := range metricasActividad {
		x := margenPDF + float64(indice)*(anchoTarjeta+separacion)

		d.fondo(colorCream100)
		d.trazo(colorLinea)
		d.pdf.SetLineWidth(0.2)
		d.pdf.RoundedRect(x, y, anchoTarjeta, altoTarjeta, 3, "1234", "FD")

		d.texto(colorLavender700)
		d.pdf.SetFont("Arial", "B", 20)
		d.pdf.SetXY(x, y+5)
		d.pdf.CellFormat(anchoTarjeta, 10, d.tr(valorATexto(fila[metrica.clave])), "", 0, "C", false, 0, "")

		d.texto(colorInk500)
		d.pdf.SetFont("Arial", "", 8.5)
		d.pdf.SetXY(x, y+16)
		d.pdf.CellFormat(anchoTarjeta, 5, d.tr(metrica.etiqueta), "", 0, "C", false, 0, "")
	}

	d.pdf.SetY(y + altoTarjeta + 8)
}

func (d *documentoReporte) tabla(reporte *dto.ReporteRespuesta) {
	columnas := columnasPorTipo[reporte.TipoReporte]
	if len(columnas) == 0 {
		return
	}

	d.texto(colorInk500)
	d.pdf.SetFont("Arial", "", 9)
	d.pdf.CellFormat(anchoUtilPDF, 5, d.tr(cantidadDeFilas(len(reporte.Datos))), "", 1, "L", false, 0, "")
	d.pdf.Ln(1)

	d.encabezadoTabla(columnas)

	for indice, fila := range reporte.Datos {
		if d.pdf.GetY()+altoFilaTabla > limiteInferior {
			d.pdf.AddPage()
			d.pdf.SetY(margenPDF)
			d.encabezadoTabla(columnas)
		}
		d.filaTabla(columnas, fila, indice%2 == 1)
	}
}

func (d *documentoReporte) encabezadoTabla(columnas []columnaPDF) {
	d.fondo(colorLavender500)
	d.texto(colorBlanco)
	d.pdf.SetFont("Arial", "B", 8.5)

	for _, columna := range columnas {
		ancho := columna.ancho * anchoUtilPDF
		etiqueta := d.recortar(columna.etiqueta, ancho-3)
		d.pdf.CellFormat(ancho, 8, d.tr(etiqueta), "", 0, columna.alinear, true, 0, "")
	}
	d.pdf.Ln(-1)
}

func (d *documentoReporte) filaTabla(columnas []columnaPDF, fila map[string]interface{}, alternada bool) {
	if alternada {
		d.fondo(colorCream100)
	} else {
		d.fondo(colorBlanco)
	}
	d.texto(colorInk900)
	d.trazo(colorLinea)
	d.pdf.SetLineWidth(0.1)
	d.pdf.SetFont("Arial", "", 8.5)

	for _, columna := range columnas {
		ancho := columna.ancho * anchoUtilPDF
		valor := d.recortar(valorATexto(fila[columna.clave]), ancho-3)
		d.pdf.CellFormat(ancho, altoFilaTabla, d.tr(valor), "B", 0, columna.alinear, true, 0, "")
	}
	d.pdf.Ln(-1)
}

func (d *documentoReporte) pie(titulo string) {
	d.pdf.SetY(-15)
	d.trazo(colorLinea)
	d.pdf.SetLineWidth(0.2)
	d.pdf.Line(margenPDF, d.pdf.GetY(), anchoPaginaA4-margenPDF, d.pdf.GetY())

	d.pdf.SetY(-12)
	d.texto(colorInk500)
	d.pdf.SetFont("Arial", "", 8)
	d.pdf.CellFormat(anchoUtilPDF/2, 6, d.tr("StemHub  ·  "+titulo), "", 0, "L", false, 0, "")
	d.pdf.CellFormat(anchoUtilPDF/2, 6, d.tr(fmt.Sprintf("Página %d de {nb}", d.pdf.PageNo())), "", 0, "R", false, 0, "")
}

func tituloReporte(tipo string) string {
	if titulo, ok := titulosReporte[tipo]; ok {
		return titulo
	}
	return tipo
}

func cantidadDeFilas(cantidad int) string {
	if cantidad == 1 {
		return "1 resultado"
	}
	return strconv.Itoa(cantidad) + " resultados"
}

// describirFiltros arma las leyendas del bloque de filtros. Los nombres de
// canción y versión se resuelven desde las propias filas del reporte (que ya
// traen código y nombre), para no pedirlos de nuevo a la base.
func describirFiltros(reporte *dto.ReporteRespuesta) []string {
	parametros := reporte.Parametros
	aplican := filtrosPorTipo[reporte.TipoReporte]
	filtros := []string{}

	if aplican.fechas {
		filtros = append(filtros, "Desde: "+fechaOSinLimite(parametros.FechaDesde), "Hasta: "+fechaOSinLimite(parametros.FechaHasta))
	}
	if aplican.cancion {
		filtros = append(filtros, "Canción: "+describirCancion(reporte))
	}
	if aplican.version {
		filtros = append(filtros, "Versión: "+describirVersion(reporte))
	}

	if reporte.TipoReporte == ReporteActividad {
		filtros = append(filtros, "Las fechas aplican a versiones y comentarios")
	}
	if len(filtros) == 0 {
		filtros = append(filtros, "Sin filtros: todo el proyecto")
	}

	if parametros.EstadoComentario != nil && strings.TrimSpace(*parametros.EstadoComentario) != "" {
		filtros = append(filtros, "Estado de comentarios: "+*parametros.EstadoComentario)
	}

	return filtros
}

func fechaOSinLimite(fecha *time.Time) string {
	if fecha == nil {
		return sinLimite
	}
	return fecha.Format(formatoFecha)
}

func describirCancion(reporte *dto.ReporteRespuesta) string {
	codigo := reporte.Parametros.CodigoCancion
	if codigo == nil {
		return "todas"
	}
	if nombre := buscarEnFilas(reporte.Datos, "codigo_cancion", *codigo, "cancion"); nombre != "" {
		return nombre
	}
	return "#" + strconv.FormatInt(*codigo, 10)
}

func describirVersion(reporte *dto.ReporteRespuesta) string {
	codigo := reporte.Parametros.CodigoVersion
	if codigo == nil {
		return "todas"
	}
	if numero := buscarEnFilas(reporte.Datos, "codigo_version", *codigo, "numero_version"); numero != "" {
		return "v" + numero
	}
	return "#" + strconv.FormatInt(*codigo, 10)
}

func buscarEnFilas(datos []map[string]interface{}, claveCodigo string, codigo int64, claveValor string) string {
	for _, fila := range datos {
		valor, ok := fila[claveCodigo]
		if !ok {
			continue
		}
		if numero, ok := valor.(int64); ok && numero == codigo {
			return valorATexto(fila[claveValor])
		}
	}
	return ""
}

func valorATexto(valor interface{}) string {
	switch dato := valor.(type) {
	case nil:
		return sinDato
	case string:
		if strings.TrimSpace(dato) == "" {
			return sinDato
		}
		return dato
	case time.Time:
		return dato.Format(formatoFecha)
	case int:
		return strconv.Itoa(dato)
	case int64:
		return strconv.FormatInt(dato, 10)
	case float64:
		return strconv.FormatFloat(dato, 'f', -1, 64)
	case bool:
		if dato {
			return "Sí"
		}
		return "No"
	default:
		return fmt.Sprintf("%v", dato)
	}
}

// nombreArchivoReporte arma algo identificable en la carpeta de descargas:
// "maqueta-solista_estado-de-canciones_2026-09-29.pdf".
func nombreArchivoReporte(reporte *dto.ReporteRespuesta) string {
	return fmt.Sprintf(
		"%s_%s_%s.pdf",
		sanitizarNombre(reporte.Proyecto.Nombre),
		sanitizarNombre(tituloReporte(reporte.TipoReporte)),
		reporte.FechaGeneracion.Local().Format("2006-01-02"),
	)
}

var acentos = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ü", "u", "Ñ", "n",
)

// sanitizarNombre deja un slug ASCII: el header Content-Disposition con
// caracteres no ASCII es ambiguo entre navegadores.
func sanitizarNombre(nombre string) string {
	nombre = acentos.Replace(strings.ToLower(strings.TrimSpace(nombre)))

	var slug strings.Builder
	guionPendiente := false
	for _, caracter := range nombre {
		switch {
		case (caracter >= 'a' && caracter <= 'z') || (caracter >= '0' && caracter <= '9'):
			if guionPendiente && slug.Len() > 0 {
				slug.WriteRune('-')
			}
			guionPendiente = false
			slug.WriteRune(caracter)
		default:
			guionPendiente = true
		}
	}

	if slug.Len() == 0 {
		return "reporte"
	}
	return slug.String()
}
