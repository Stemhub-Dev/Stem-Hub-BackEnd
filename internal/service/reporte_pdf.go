package service

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
)

// El PDF sigue el design system del frontend
// (stemhub-frontend/src/styles/tokens.css) y el lenguaje visual de la landing
// (portadas geométricas, onda decorativa): tiene que leerse como una pieza de
// StemHub, no como un volcado de datos.
type colorRGB struct{ r, g, b int }

// colorHex convierte "#rrggbb", así los tokens se copian tal cual.
func colorHex(hex string) colorRGB {
	valor, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil {
		return colorRGB{}
	}
	return colorRGB{int(valor >> 16 & 0xff), int(valor >> 8 & 0xff), int(valor & 0xff)}
}

var (
	colorLavender50  = colorHex("#f6f5fa")
	colorLavender500 = colorHex("#9b8ec4")
	colorLavender700 = colorHex("#6f62a0")
	colorCoral50     = colorHex("#fdebe5")
	colorCoral700    = colorHex("#a8462a")
	colorTeal50      = colorHex("#e2f1ee")
	colorTeal700     = colorHex("#2c6f65")
	colorCream100    = colorHex("#f7f5f2")
	colorCream200    = colorHex("#efebe6")
	colorLinea       = colorHex("#e6e1da")
	colorInk400      = colorHex("#737080")
	colorInk500      = colorHex("#6e6b7a")
	colorInk600      = colorHex("#5c5968")
	colorInk700      = colorHex("#4a4756")
	colorInk900      = colorHex("#2c2a35")
	colorSuccess50   = colorHex("#edf5f0")
	colorSuccess700  = colorHex("#3f6b52")
	colorWarning50   = colorHex("#fbf3e2")
	colorWarning700  = colorHex("#8a5f10")
	colorBlanco      = colorHex("#ffffff")
)

const (
	margenPDF      = 15.0
	anchoPaginaA4  = 210.0
	altoPaginaA4   = 297.0
	anchoUtilPDF   = anchoPaginaA4 - (margenPDF * 2)
	bordeDerecho   = anchoPaginaA4 - margenPDF
	limiteInferior = altoPaginaA4 - 22
	sinDato        = "-"
	formatoFecha   = "02/01/2006"

	altoPortada      = 90.0
	altoEncabezadoPg = 18.0
	altoFilaCancion  = 16.0

	// Columnas de la lista de canciones (posición x en mm).
	xNumero      = margenPDF
	xMiniPortada = margenPDF + 11
	xCancion     = margenPDF + 26
	anchoCancion = 72.0
	xComentarios = 120.0
	anchoBarra   = 40.0
)

// Títulos legibles: el slug del tipo ("estado-canciones") sirve para la URL,
// no para mostrárselo a una persona.
var titulosReporte = map[string]string{
	ReporteEstadoCanciones: "Estado del proyecto",
}

// --- Portadas generadas -----------------------------------------------------

// Mismas portadas geométricas que los proyectos de ejemplo de la landing
// (stemhub-frontend/src/features/landing/portadasEjemplo.tsx), en una caja de
// 100x100. El proyecto recibe siempre la misma según su nombre: el reporte
// tiene identidad visual sin depender de imágenes.
type formaPortada struct {
	tipo  byte // 'c' círculo (cx, cy, r) · 'r' rectángulo (x, y, w, h) · 'p' polígono (x1, y1, ...)
	color colorRGB
	v     []float64
}

type disenoPortada struct {
	fondo  colorRGB
	formas []formaPortada
}

var disenosPortada = []disenoPortada{
	{colorHex("#d1495b"), []formaPortada{
		{'c', colorHex("#ffe3b3"), []float64{68, 34, 17}},
		{'r', colorHex("#5b2a5e"), []float64{0, 68, 100, 32}},
	}},
	{colorHex("#2c6f65"), []formaPortada{{'c', colorHex("#e2f1ee"), []float64{50, 118, 62}}}},
	{colorHex("#efedf6"), []formaPortada{{'r', colorHex("#3a3359"), []float64{30, 30, 40, 40}}}},
	{colorHex("#f2a65a"), []formaPortada{{'p', colorHex("#a8462a"), []float64{0, 100, 100, 0, 100, 100}}}},
	{colorHex("#6f62a0"), []formaPortada{{'c', colorHex("#f6f5fa"), []float64{50, 50, 27}}}},
	{colorHex("#2a2540"), []formaPortada{{'r', colorHex("#f2a65a"), []float64{0, 46, 100, 8}}}},
	{colorHex("#3a3359"), []formaPortada{{'c', colorHex("#f2a65a"), []float64{50, 100, 34}}}},
	{colorHex("#fdebe5"), []formaPortada{
		{'r', colorHex("#a8462a"), []float64{24, 24, 52, 12}},
		{'r', colorHex("#d1495b"), []float64{24, 44, 52, 12}},
		{'r', colorHex("#f2a65a"), []float64{24, 64, 52, 12}},
	}},
}

// indicePortada elige un diseño estable a partir del nombre.
func indicePortada(nombre string) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(strings.ToLower(strings.TrimSpace(nombre))))
	return int(hash.Sum32() % uint32(len(disenosPortada)))
}

// --- Datos agregados --------------------------------------------------------

// totalesProyecto agrega las filas por canción. Mantener en sync con
// resumirEstadoProyecto del frontend (features/reports/columnasReporte.ts).
type totalesProyecto struct {
	canciones, versiones, comentarios, resueltos, conPendientes int64
}

func totalesEstado(datos []map[string]interface{}) totalesProyecto {
	totales := totalesProyecto{canciones: int64(len(datos))}
	for _, fila := range datos {
		totales.versiones += enteroDe(fila["total_versiones"])
		totales.comentarios += enteroDe(fila["comentarios"])
		totales.resueltos += enteroDe(fila["comentarios_resueltos"])
		if enteroDe(fila["comentarios_pendientes"]) > 0 {
			totales.conPendientes++
		}
	}
	return totales
}

// porcentajeResueltos redondea hacia abajo, igual que el frontend: un 99,6%
// no se muestra como 100% si queda un comentario abierto.
func (t totalesProyecto) porcentajeResueltos() string {
	if t.comentarios == 0 {
		return sinDato
	}
	return strconv.FormatInt(t.resueltos*100/t.comentarios, 10) + "%"
}

func enteroDe(valor interface{}) int64 {
	switch dato := valor.(type) {
	case int64:
		return dato
	case int:
		return int64(dato)
	case float64:
		return int64(dato)
	default:
		return 0
	}
}

// etiquetaEstado es la píldora de cada canción: lo primero que se mira para
// saber qué necesita atención.
type etiquetaEstado struct {
	texto        string
	fondo, color colorRGB
}

func estadoCancion(fila map[string]interface{}) etiquetaEstado {
	pendientes := enteroDe(fila["comentarios_pendientes"])
	switch {
	case enteroDe(fila["ultima_version"]) <= 0:
		return etiquetaEstado{"Sin versiones", colorCream200, colorInk600}
	case pendientes > 0:
		return etiquetaEstado{plural(pendientes, "pendiente", "pendientes"), colorWarning50, colorWarning700}
	case enteroDe(fila["comentarios"]) == 0:
		return etiquetaEstado{"Sin revisar", colorLavender50, colorLavender700}
	default:
		return etiquetaEstado{"Al día", colorSuccess50, colorSuccess700}
	}
}

func plural(cantidad int64, singular, varios string) string {
	if cantidad == 1 {
		return "1 " + singular
	}
	return strconv.FormatInt(cantidad, 10) + " " + varios
}

// --- Documento --------------------------------------------------------------

// documentoReporte envuelve el fpdf con el traductor de codificación: las
// fuentes core (Helvetica) son cp1252, sin traducir "Canción" sale con basura.
type documentoReporte struct {
	pdf     *fpdf.Fpdf
	tr      func(string) string
	reporte *dto.ReporteRespuesta
	titulo  string
	// conPortadaReal indica que la imagen del proyecto está registrada como
	// nombreImagenPortada.
	conPortadaReal bool
	// colores de la banda del proyecto, derivados de su portada como en la
	// tarjeta "Retomá tu sesión" del Inicio.
	colores fondoPortada
}

const nombreImagenPortada = "portada-proyecto"

func (d *documentoReporte) fondo(color colorRGB) { d.pdf.SetFillColor(color.r, color.g, color.b) }
func (d *documentoReporte) texto(color colorRGB) { d.pdf.SetTextColor(color.r, color.g, color.b) }
func (d *documentoReporte) trazo(color colorRGB) { d.pdf.SetDrawColor(color.r, color.g, color.b) }
func (d *documentoReporte) ancho(texto string) float64 {
	return d.pdf.GetStringWidth(d.tr(texto))
}

func (d *documentoReporte) escribir(x, y, ancho, alto float64, texto, alinear string) {
	d.pdf.SetXY(x, y)
	d.pdf.CellFormat(ancho, alto, d.tr(texto), "", 0, alinear, false, 0, "")
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

// portada es la imagen real del proyecto ya preparada como JPEG (ver
// reporte_portada.go); nil usa la portada generada.
func construirReportePDF(reporte *dto.ReporteRespuesta, portada *portadaPDF) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	doc := &documentoReporte{
		pdf:     pdf,
		tr:      pdf.UnicodeTranslatorFromDescriptor(""),
		reporte: reporte,
		titulo:  tituloReporte(reporte.TipoReporte),
		// Sin portada real, los colores salen del fondo de la portada
		// generada: la tapa y la banda siempre combinan.
		colores: fondoDesdeColor(disenosPortada[indicePortada(reporte.Proyecto.Nombre)].fondo),
	}

	pdf.SetTitle(doc.titulo+" - "+reporte.Proyecto.Nombre, true)
	pdf.SetAuthor("StemHub", true)
	pdf.SetMargins(margenPDF, margenPDF, margenPDF)
	// Sin margen interno de celda: los textos alinean al milímetro con las
	// barras, las portadas y el borde de las píldoras.
	pdf.SetCellMargin(0)
	// Los saltos de página los maneja la lista a mano, para no cortar una
	// canción al medio y repetir el encabezado en la hoja siguiente.
	pdf.SetAutoPageBreak(false, 0)
	pdf.AliasNbPages("")
	pdf.SetFooterFunc(doc.pie)

	if portada != nil && len(portada.jpeg) > 0 {
		pdf.RegisterImageOptionsReader(nombreImagenPortada, fpdf.ImageOptions{ImageType: "JPG"}, bytes.NewReader(portada.jpeg))
		doc.conPortadaReal = pdf.Ok()
		if !doc.conPortadaReal {
			// Los errores de fpdf quedan pegados al documento: se descarta el
			// intento y se arma de nuevo con la portada generada.
			return construirReportePDF(reporte, nil)
		}
		doc.colores = portada.colores
	}

	pdf.AddPage()
	doc.portada()
	y := doc.indicadores(altoPortada + 10)
	doc.listaCanciones(y + 12)

	var contenido bytes.Buffer
	if err := pdf.Output(&contenido); err != nil {
		return nil, err
	}
	return contenido.Bytes(), nil
}

// portada: la "tapa del disco" del proyecto. Igual que "Retomá tu sesión"
// del Inicio, el fondo es un pastel del tono de la portada, el texto va en
// tinta oscura y los detalles en el acento de ese mismo tono.
func (d *documentoReporte) portada() {
	proyecto := d.reporte.Proyecto
	colores := d.colores

	d.fondo(colores.fondo)
	d.pdf.Rect(0, 0, anchoPaginaA4, altoPortada, "F")
	d.onda(0, altoPortada-11, anchoPaginaA4, 14, colores.acento, 0.10)
	d.marca(bordeDerecho, 12, colorInk900)

	const tamanoTapa = 54.0
	yTapa := 20.0
	d.sombra(margenPDF, yTapa, tamanoTapa, tamanoTapa, 3)
	d.tapaProyecto(margenPDF, yTapa, tamanoTapa, 3)

	x := margenPDF + tamanoTapa + 9
	anchoTexto := bordeDerecho - x

	d.texto(colores.acento)
	d.pdf.SetFont("Helvetica", "B", 8)
	d.escribir(x, yTapa+1, anchoTexto, 4, strings.ToUpper(d.titulo), "L")

	// El nombre achica la letra antes de recortarse: es el dato principal.
	tamano := 26.0
	for ; tamano > 16; tamano-- {
		d.pdf.SetFont("Helvetica", "B", tamano)
		if d.ancho(proyecto.Nombre) <= anchoTexto {
			break
		}
	}
	d.texto(colorInk900)
	d.escribir(x, yTapa+6, anchoTexto, tamano*0.45, d.recortar(proyecto.Nombre, anchoTexto), "L")

	d.texto(colorInk700)
	d.pdf.SetFont("Helvetica", "", 11)
	detalle := strings.TrimSpace(proyecto.Tipo)
	if detalle != "" {
		detalle += "  ·  "
	}
	detalle += plural(int64(len(d.reporte.Datos)), "canción", "canciones")
	d.escribir(x, yTapa+20, anchoTexto, 6, detalle, "L")

	if estado := strings.TrimSpace(proyecto.Estado); estado != "" {
		// El acento tiene contraste AA con el fondo pastel, así que también
		// lo tiene con el texto blanco.
		d.pastilla(x, yTapa+30, estado, colores.acento, colorBlanco)
	}

	generado := d.reporte.FechaGeneracion.Local()
	d.texto(colorInk600)
	d.pdf.SetFont("Helvetica", "", 8)
	d.escribir(x, yTapa+41, anchoTexto, 4,
		"Generado el "+generado.Format(formatoFecha)+" a las "+generado.Format("15:04 (UTC-07:00)"), "L")
}

// indicadores: las tres cifras del proyecto, con los acentos del Inicio
// (un color por indicador). Devuelve dónde terminan.
func (d *documentoReporte) indicadores(y float64) float64 {
	totales := totalesEstado(d.reporte.Datos)

	detallePendientes := "Todo al día"
	if totales.conPendientes > 0 {
		detallePendientes = "Necesitan una revisión"
	}
	progreso := -1.0
	if totales.comentarios > 0 {
		progreso = float64(totales.resueltos) / float64(totales.comentarios)
	}

	tarjetas := []struct {
		etiqueta, valor, detalle string
		fondo, acento            colorRGB
		progreso                 float64
	}{
		{"CANCIONES", strconv.FormatInt(totales.canciones, 10),
			plural(totales.versiones, "versión en total", "versiones en total"), colorLavender50, colorLavender700, -1},
		{"COMENTARIOS RESUELTOS", totales.porcentajeResueltos(),
			fmt.Sprintf("%d de %d comentarios", totales.resueltos, totales.comentarios), colorTeal50, colorTeal700, progreso},
		{"CON PENDIENTES", strconv.FormatInt(totales.conPendientes, 10), detallePendientes, colorCoral50, colorCoral700, -1},
	}

	const separacion, alto = 5.0, 36.0
	anchoTarjeta := (anchoUtilPDF - separacion*float64(len(tarjetas)-1)) / float64(len(tarjetas))

	for indice, tarjeta := range tarjetas {
		x := margenPDF + float64(indice)*(anchoTarjeta+separacion)
		interior := anchoTarjeta - 12

		d.fondo(tarjeta.fondo)
		d.pdf.RoundedRect(x, y, anchoTarjeta, alto, 3, "1234", "F")

		d.texto(tarjeta.acento)
		d.pdf.SetFont("Helvetica", "B", 7)
		d.escribir(x+6, y+6, interior, 4, d.recortar(tarjeta.etiqueta, interior), "L")

		d.pdf.SetFont("Helvetica", "B", 24)
		d.escribir(x+6, y+11, interior, 11, tarjeta.valor, "L")

		if tarjeta.progreso >= 0 {
			d.barra(x+6, y+24.5, interior, 1.8, tarjeta.progreso, colorBlanco, tarjeta.acento)
		}

		d.texto(colorInk600)
		d.pdf.SetFont("Helvetica", "", 8)
		d.escribir(x+6, y+28, interior, 4, d.recortar(tarjeta.detalle, interior), "L")
	}

	return y + alto
}

// listaCanciones: el tracklist del proyecto, una fila por canción.
func (d *documentoReporte) listaCanciones(y float64) {
	d.texto(colorInk900)
	d.pdf.SetFont("Helvetica", "B", 15)
	d.escribir(margenPDF, y, anchoUtilPDF/2, 7, "Canciones", "L")

	subtitulo := "Las que necesitan atención, primero"
	if filtros := describirFiltros(d.reporte); len(filtros) > 0 {
		subtitulo = strings.Join(filtros, "  ·  ")
	}
	d.texto(colorInk500)
	d.pdf.SetFont("Helvetica", "", 8.5)
	d.escribir(margenPDF+anchoUtilPDF/2, y+1, anchoUtilPDF/2, 6, d.recortar(subtitulo, anchoUtilPDF/2), "R")

	y += 12
	if len(d.reporte.Datos) == 0 {
		d.estadoVacio(y)
		return
	}

	y = d.encabezadoLista(y)
	for indice, fila := range d.reporte.Datos {
		if y+altoFilaCancion > limiteInferior {
			d.pdf.AddPage()
			y = d.encabezadoLista(d.encabezadoContinuacion())
		}
		d.filaCancion(indice+1, fila, y)
		y += altoFilaCancion
	}
}

func (d *documentoReporte) encabezadoLista(y float64) float64 {
	d.texto(colorInk400)
	d.pdf.SetFont("Helvetica", "B", 7)
	d.escribir(xNumero, y, 10, 4, "#", "L")
	d.escribir(xCancion, y, anchoCancion, 4, "CANCIÓN", "L")
	d.escribir(xComentarios, y, anchoBarra, 4, "COMENTARIOS", "L")
	d.escribir(bordeDerecho-30, y, 30, 4, "ESTADO", "R")

	d.trazo(colorLinea)
	d.pdf.SetLineWidth(0.3)
	d.pdf.Line(margenPDF, y+6, bordeDerecho, y+6)
	return y + 7
}

func (d *documentoReporte) filaCancion(numero int, fila map[string]interface{}, y float64) {
	nombre := valorATexto(fila["cancion"])
	ultima := enteroDe(fila["ultima_version"])
	comentarios := enteroDe(fila["comentarios"])
	resueltos := enteroDe(fila["comentarios_resueltos"])

	d.texto(colorLavender500)
	d.pdf.SetFont("Helvetica", "B", 12)
	d.escribir(xNumero, y, 10, altoFilaCancion, fmt.Sprintf("%02d", numero), "L")

	// Las canciones no tienen portada propia: llevan la del proyecto, como
	// los temas de un disco.
	d.tapaProyecto(xMiniPortada, y+2.5, 11, 1.5)

	d.texto(colorInk900)
	d.pdf.SetFont("Helvetica", "B", 10.5)
	d.escribir(xCancion, y+3, anchoCancion, 5, d.recortar(nombre, anchoCancion), "L")

	versiones := "Todavía sin versiones"
	if ultima > 0 {
		versiones = "Versión actual " + versionATexto(ultima) + "  ·  " +
			plural(enteroDe(fila["total_versiones"]), "versión", "versiones")
	}
	d.texto(colorInk500)
	d.pdf.SetFont("Helvetica", "", 8)
	d.escribir(xCancion, y+8.5, anchoCancion, 4, d.recortar(versiones, anchoCancion), "L")

	if comentarios > 0 {
		d.texto(colorInk600)
		d.pdf.SetFont("Helvetica", "", 7.5)
		d.escribir(xComentarios, y+3.5, anchoBarra, 4, fmt.Sprintf("%d de %d resueltos", resueltos, comentarios), "L")
		d.barra(xComentarios, y+9, anchoBarra, 2, float64(resueltos)/float64(comentarios), colorCream200, colorTeal700)
	} else {
		d.texto(colorInk400)
		d.pdf.SetFont("Helvetica", "", 8)
		d.escribir(xComentarios, y, anchoBarra, altoFilaCancion, "Sin comentarios", "L")
	}

	estado := estadoCancion(fila)
	d.pdf.SetFont("Helvetica", "B", 7.5)
	d.pastilla(bordeDerecho-(d.ancho(estado.texto)+8), y+5, estado.texto, estado.fondo, estado.color)

	d.trazo(colorCream200)
	d.pdf.SetLineWidth(0.2)
	d.pdf.Line(margenPDF, y+altoFilaCancion, bordeDerecho, y+altoFilaCancion)
}

func (d *documentoReporte) estadoVacio(y float64) {
	titulo := "Este proyecto todavía no tiene canciones"
	detalle := "Cuando subas la primera versión, vas a ver acá cómo avanza."
	if d.reporte.Parametros.CodigoCancion != nil {
		titulo = "No hay canciones para el filtro elegido"
		detalle = "Probá con \"Todas las canciones\" para ver el proyecto completo."
	}

	d.fondo(colorCream100)
	d.pdf.RoundedRect(margenPDF, y, anchoUtilPDF, 30, 3, "1234", "F")
	d.texto(colorInk900)
	d.pdf.SetFont("Helvetica", "B", 11)
	d.escribir(margenPDF, y+9, anchoUtilPDF, 6, titulo, "C")
	d.texto(colorInk500)
	d.pdf.SetFont("Helvetica", "", 8.5)
	d.escribir(margenPDF, y+16, anchoUtilPDF, 5, detalle, "C")
}

// encabezadoContinuacion: franja angosta en las hojas 2+, para que una hoja
// suelta siga diciendo de qué proyecto es.
func (d *documentoReporte) encabezadoContinuacion() float64 {
	d.fondo(d.colores.fondo)
	d.pdf.Rect(0, 0, anchoPaginaA4, altoEncabezadoPg, "F")

	d.marca(margenPDF+22, altoEncabezadoPg/2, colorInk900)

	d.texto(d.colores.acento)
	d.pdf.SetFont("Helvetica", "", 8.5)
	texto := d.reporte.Proyecto.Nombre + "  ·  " + d.titulo
	d.escribir(margenPDF+anchoUtilPDF/3, altoEncabezadoPg/2-2, anchoUtilPDF*2/3, 4,
		d.recortar(texto, anchoUtilPDF*2/3), "R")

	return altoEncabezadoPg + 8
}

func (d *documentoReporte) pie() {
	d.trazo(colorLinea)
	d.pdf.SetLineWidth(0.2)
	d.pdf.Line(margenPDF, altoPaginaA4-15, bordeDerecho, altoPaginaA4-15)

	d.texto(colorInk500)
	d.pdf.SetFont("Helvetica", "", 7.5)
	izquierda := d.recortar("StemHub  ·  "+d.titulo+"  ·  "+d.reporte.Proyecto.Nombre, anchoUtilPDF*0.7)
	d.escribir(margenPDF, altoPaginaA4-13, anchoUtilPDF*0.7, 6, izquierda, "L")
	d.escribir(margenPDF+anchoUtilPDF*0.7, altoPaginaA4-13, anchoUtilPDF*0.3, 6,
		fmt.Sprintf("Página %d de {nb}", d.pdf.PageNo()), "R")
}

// --- Piezas gráficas --------------------------------------------------------

// marca: el logotipo "StemHub" con una mini onda, alineado a la derecha de x
// y centrado verticalmente en y.
func (d *documentoReporte) marca(xDerecha, y float64, color colorRGB) {
	d.pdf.SetFont("Helvetica", "B", 10)
	anchoTexto := d.ancho("StemHub")
	d.texto(color)
	d.escribir(xDerecha-anchoTexto, y-2.5, anchoTexto, 5, "StemHub", "R")

	altos := []float64{2, 4, 3, 5, 2.5}
	x := xDerecha - anchoTexto - 2 - float64(len(altos))*1.3
	d.fondo(color)
	for indice, alto := range altos {
		d.pdf.RoundedRect(x+float64(indice)*1.3, y-alto/2, 0.7, alto, 0.35, "1234", "F")
	}
}

// tapaProyecto dibuja la portada del proyecto: la real si se pudo leer, o la
// generada. fpdf embebe la imagen una sola vez aunque se dibuje en cada fila.
func (d *documentoReporte) tapaProyecto(x, y, tamano, radio float64) {
	if !d.conPortadaReal {
		d.portadaGenerada(indicePortada(d.reporte.Proyecto.Nombre), x, y, tamano, radio)
		return
	}
	d.pdf.ClipRoundedRect(x, y, tamano, tamano, radio, false)
	d.pdf.ImageOptions(nombreImagenPortada, x, y, tamano, tamano, false,
		fpdf.ImageOptions{ImageType: "JPG"}, 0, "")
	d.pdf.ClipEnd()
}

// sombra: la sombra suave bajo la tapa, como la portada de "Retomá tu
// sesión" (0 8px 20px rgba(0,0,0,.18)). fpdf no tiene desenfoque: se
// aproxima con capas translúcidas cada vez más grandes.
func (d *documentoReporte) sombra(x, y, ancho, alto, radio float64) {
	const capas = 8
	d.fondo(colorInk900)
	d.pdf.SetAlpha(0.025, "Normal")
	for capa := capas; capa >= 1; capa-- {
		expansion := float64(capa) * 0.6
		d.pdf.RoundedRect(x-expansion, y+2.5-expansion, ancho+2*expansion, alto+2*expansion,
			radio+expansion, "1234", "F")
	}
	d.pdf.SetAlpha(1, "Normal")
}

// onda: la forma de onda decorativa de la landing (OndaDecorativa.tsx).
func (d *documentoReporte) onda(x, yCentro, ancho, altoMaximo float64, color colorRGB, opacidad float64) {
	barras := int(ancho / 2.8)
	paso := ancho / float64(barras)
	d.pdf.SetAlpha(opacidad, "Normal")
	d.fondo(color)
	for indice := 0; indice < barras; indice++ {
		posicion := float64(indice)
		alto := altoMaximo * (0.2 + 0.8*math.Abs(math.Sin(posicion*0.55)*math.Cos(posicion*0.19)))
		d.pdf.RoundedRect(x+posicion*paso, yCentro-alto/2, paso*0.5, alto, paso*0.25, "1234", "F")
	}
	d.pdf.SetAlpha(1, "Normal")
}

func (d *documentoReporte) portadaGenerada(indice int, x, y, tamano, radio float64) {
	diseno := disenosPortada[indice]
	escala := tamano / 100

	d.pdf.ClipRoundedRect(x, y, tamano, tamano, radio, false)
	d.fondo(diseno.fondo)
	d.pdf.Rect(x, y, tamano, tamano, "F")
	for _, forma := range diseno.formas {
		d.fondo(forma.color)
		v := forma.v
		switch forma.tipo {
		case 'c':
			d.pdf.Circle(x+v[0]*escala, y+v[1]*escala, v[2]*escala, "F")
		case 'r':
			d.pdf.Rect(x+v[0]*escala, y+v[1]*escala, v[2]*escala, v[3]*escala, "F")
		case 'p':
			puntos := make([]fpdf.PointType, 0, len(v)/2)
			for i := 0; i+1 < len(v); i += 2 {
				puntos = append(puntos, fpdf.PointType{X: x + v[i]*escala, Y: y + v[i+1]*escala})
			}
			d.pdf.Polygon(puntos, "F")
		}
	}
	d.pdf.ClipEnd()
}

func (d *documentoReporte) barra(x, y, ancho, alto, progreso float64, pista, relleno colorRGB) {
	d.fondo(pista)
	d.pdf.RoundedRect(x, y, ancho, alto, alto/2, "1234", "F")
	progreso = math.Max(0, math.Min(1, progreso))
	if lleno := ancho * progreso; lleno >= alto {
		d.fondo(relleno)
		d.pdf.RoundedRect(x, y, lleno, alto, alto/2, "1234", "F")
	}
}

func (d *documentoReporte) pastilla(x, y float64, texto string, fondo, color colorRGB) {
	d.pdf.SetFont("Helvetica", "B", 7.5)
	ancho := d.ancho(texto) + 8
	d.fondo(fondo)
	d.pdf.RoundedRect(x, y, ancho, 6, 3, "1234", "F")
	d.texto(color)
	d.escribir(x, y, ancho, 6, texto, "C")
}

// --- Textos -----------------------------------------------------------------

func tituloReporte(tipo string) string {
	if titulo, ok := titulosReporte[tipo]; ok {
		return titulo
	}
	return tipo
}

// describirFiltros lista solo los filtros activos: sin filtros, el reporte es
// el proyecto completo y no hace falta aclararlo. El nombre de la canción se
// resuelve desde las propias filas (ya traen código y nombre), para no
// pedirlo de nuevo a la base.
func describirFiltros(reporte *dto.ReporteRespuesta) []string {
	filtros := []string{}
	if reporte.Parametros.CodigoCancion != nil {
		filtros = append(filtros, "Canción: "+describirCancion(reporte))
	}
	estado := reporte.Parametros.EstadoComentario
	if estado != nil && strings.TrimSpace(*estado) != "" {
		filtros = append(filtros, "Comentarios: "+*estado)
	}
	return filtros
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

func buscarEnFilas(datos []map[string]interface{}, claveCodigo string, codigo int64, claveValor string) string {
	for _, fila := range datos {
		if numero, ok := fila[claveCodigo].(int64); ok && numero == codigo {
			return valorATexto(fila[claveValor])
		}
	}
	return ""
}

// versionATexto muestra un número de versión como en el resto de la app
// ("v3"). Las filas vienen de rows.Scan en int64; ultima_version es 0 en una
// canción sin versiones.
func versionATexto(valor interface{}) string {
	numero, ok := valor.(int64)
	if !ok {
		return valorATexto(valor)
	}
	if numero <= 0 {
		return sinDato
	}
	return dto.EtiquetaVersion(numero)
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
// "maqueta-solista_estado-del-proyecto_2026-09-29.pdf".
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
