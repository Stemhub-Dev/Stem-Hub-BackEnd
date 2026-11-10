package service

import (
	"image"
	"math"

	xdraw "golang.org/x/image/draw"
)

// Port de stemhub-frontend/src/lib/colorPortada.ts: los colores que el
// Inicio ("Retomá tu sesión") deriva de la portada de un proyecto. El PDF
// usa el mismo algoritmo y los mismos parámetros para que un proyecto tenga
// el mismo color en la app y en el reporte. Mantener los dos en sync.

// fondoPortada son los dos colores derivados de la portada.
type fondoPortada struct {
	// fondo es pastel: el texto va encima en tinta oscura.
	fondo colorRGB
	// acento es el tono oscuro, legible (AA) sobre el fondo.
	acento colorRGB
}

type colorHSL struct{ h, s, l float64 }

const (
	// Lado de la miniatura que se analiza, como LADO_MUESTRA del frontend.
	ladoMuestraColor = 24

	franjasTono             = 36
	intensidadMinima        = 0.08
	fraccionCromaticaMinima = 0.1

	contrasteMinimoAcento    = 4.5
	saturacionMaximaFondo    = 0.5
	luminosidadFondo         = 0.94
	luminosidadAcentoInicial = 0.42
)

func rgbAHsl(c colorRGB) colorHSL {
	rn, gn, bn := float64(c.r)/255, float64(c.g)/255, float64(c.b)/255
	maximo := math.Max(rn, math.Max(gn, bn))
	minimo := math.Min(rn, math.Min(gn, bn))
	l := (maximo + minimo) / 2
	delta := maximo - minimo
	if delta == 0 {
		return colorHSL{0, 0, l}
	}

	s := delta / (1 - math.Abs(2*l-1))
	var h float64
	switch maximo {
	case rn:
		h = math.Mod((gn-bn)/delta, 6)
	case gn:
		h = (bn-rn)/delta + 2
	default:
		h = (rn-gn)/delta + 4
	}
	return colorHSL{math.Mod(h*60+360, 360), s, l}
}

func hslARgb(c colorHSL) colorRGB {
	croma := (1 - math.Abs(2*c.l-1)) * c.s
	x := croma * (1 - math.Abs(math.Mod(c.h/60, 2)-1))
	m := c.l - croma/2

	var r, g, b float64
	switch {
	case c.h < 60:
		r, g, b = croma, x, 0
	case c.h < 120:
		r, g, b = x, croma, 0
	case c.h < 180:
		r, g, b = 0, croma, x
	case c.h < 240:
		r, g, b = 0, x, croma
	case c.h < 300:
		r, g, b = x, 0, croma
	default:
		r, g, b = croma, 0, x
	}
	// math.Round coincide con Math.round de JS para valores positivos.
	return colorRGB{
		int(math.Round((r + m) * 255)),
		int(math.Round((g + m) * 255)),
		int(math.Round((b + m) * 255)),
	}
}

// luminanciaRelativa según WCAG 2.x.
func luminanciaRelativa(c colorRGB) float64 {
	canal := func(valor int) float64 {
		v := float64(valor) / 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*canal(c.r) + 0.7152*canal(c.g) + 0.0722*canal(c.b)
}

func contrasteEntre(a, b colorRGB) float64 {
	la, lb := luminanciaRelativa(a), luminanciaRelativa(b)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}

type pixelLeido struct {
	color     colorRGB
	franja    int
	cromatico bool
}

func promedioPixeles(pixeles []pixelLeido) (colorRGB, bool) {
	if len(pixeles) == 0 {
		return colorRGB{}, false
	}
	var r, g, b float64
	for _, pixel := range pixeles {
		r += float64(pixel.color.r)
		g += float64(pixel.color.g)
		b += float64(pixel.color.b)
	}
	n := float64(len(pixeles))
	return colorRGB{int(math.Round(r / n)), int(math.Round(g / n)), int(math.Round(b / n))}, true
}

// colorDominante elige el tono que más área ocupa (histograma de tonos) y
// promedia solo los píxeles de ese tono: así gana el rosa de un fondo grande
// y no lo arrastran unos detalles rojos muy saturados. Los píxeles casi
// transparentes no cuentan; false si no hay ninguno visible. pixeles es
// RGBA sin premultiplicar, como getImageData en el navegador.
func colorDominante(pixeles []uint8) (colorRGB, bool) {
	leidos, votos := leerPixeles(pixeles)

	cromaticos := []pixelLeido{}
	for _, pixel := range leidos {
		if pixel.cromatico {
			cromaticos = append(cromaticos, pixel)
		}
	}
	if float64(len(cromaticos)) < float64(len(leidos))*fraccionCromaticaMinima {
		return promedioPixeles(leidos)
	}

	centro := franjaCentral(votos)
	cercanos := []pixelLeido{}
	for _, pixel := range cromaticos {
		if distanciaFranjas(pixel.franja, centro) <= 1 {
			cercanos = append(cercanos, pixel)
		}
	}
	return promedioPixeles(cercanos)
}

// leerPixeles clasifica los píxeles visibles por franja de tono y suma los
// votos de los que tienen color. Manda el área; la intensidad solo
// desempata un poco.
func leerPixeles(pixeles []uint8) ([]pixelLeido, []float64) {
	leidos := []pixelLeido{}
	votos := make([]float64, franjasTono)
	for i := 0; i+3 < len(pixeles); i += 4 {
		if pixeles[i+3] < 128 {
			continue
		}
		c := colorRGB{int(pixeles[i]), int(pixeles[i+1]), int(pixeles[i+2])}
		hsl := rgbAHsl(c)
		// Saturación "percibida": un blanco o un negro puros no aportan tono
		// aunque su HSL diga saturación alta.
		intensidad := hsl.s * (1 - math.Abs(2*hsl.l-1))
		franja := int(math.Floor(hsl.h/(360/franjasTono))) % franjasTono
		cromatico := intensidad >= intensidadMinima
		if cromatico {
			votos[franja] += 1 + intensidad
		}
		leidos = append(leidos, pixelLeido{c, franja, cromatico})
	}
	return leidos, votos
}

func normalizarFranja(franja int) int {
	return (franja%franjasTono + franjasTono) % franjasTono
}

func distanciaFranjas(a, b int) int {
	diferencia := a - b
	if diferencia < 0 {
		diferencia = -diferencia
	}
	return min(diferencia, franjasTono-diferencia)
}

// franjaCentral busca la franja con más votos sumando sus vecinas (un tono
// justo en el borde entre dos franjas no pierde contra uno concentrado en
// una) y devuelve, dentro de esa vecindad, la franja con más votos propios:
// si se tomara el centro de la vecindad, un tono vecino (el rojo al lado del
// rosa) entraría en el promedio y lo correría.
func franjaCentral(votos []float64) int {
	vecindad := func(franja int) float64 {
		return votos[normalizarFranja(franja-1)] + votos[franja] + votos[normalizarFranja(franja+1)]
	}
	ganadora := 0
	for franja := 1; franja < franjasTono; franja++ {
		if vecindad(franja) > vecindad(ganadora) {
			ganadora = franja
		}
	}

	centro := ganadora
	for _, franja := range []int{normalizarFranja(ganadora - 1), normalizarFranja(ganadora + 1)} {
		if votos[franja] > votos[centro] {
			centro = franja
		}
	}
	return centro
}

// fondoDesdeColor arma un fondo pastel del mismo tono y un acento oscurecido
// hasta que se lea bien sobre ese fondo. Un color gris o negro queda en un
// gris neutro (saturación 0).
func fondoDesdeColor(c colorRGB) fondoPortada {
	hsl := rgbAHsl(c)
	saturacion := math.Min(saturacionMaximaFondo, hsl.s)
	fondo := hslARgb(colorHSL{hsl.h, saturacion, luminosidadFondo})

	luminosidad := luminosidadAcentoInicial
	acento := hslARgb(colorHSL{hsl.h, saturacion, luminosidad})
	for contrasteEntre(acento, fondo) < contrasteMinimoAcento && luminosidad > 0.1 {
		luminosidad -= 0.02
		acento = hslARgb(colorHSL{hsl.h, saturacion, luminosidad})
	}
	return fondoPortada{fondo: fondo, acento: acento}
}

// muestraParaColor reduce la imagen completa a 24x24, como hace el Inicio al
// dibujarla en un canvas, y devuelve sus píxeles RGBA sin premultiplicar.
func muestraParaColor(origen image.Image) []uint8 {
	muestra := image.NewNRGBA(image.Rect(0, 0, ladoMuestraColor, ladoMuestraColor))
	xdraw.ApproxBiLinear.Scale(muestra, muestra.Bounds(), origen, origen.Bounds(), xdraw.Src, nil)
	return muestra.Pix
}

// fondoDeImagen devuelve los colores de una portada; false si no tiene
// píxeles visibles.
func fondoDeImagen(origen image.Image) (fondoPortada, bool) {
	dominante, ok := colorDominante(muestraParaColor(origen))
	if !ok {
		return fondoPortada{}, false
	}
	return fondoDesdeColor(dominante), true
}
