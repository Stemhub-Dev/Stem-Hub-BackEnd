package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // registra el decodificador PNG para image.Decode
	"log"
	"time"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registra el decodificador WebP para image.Decode

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
)

const (
	// Lado en píxeles de la portada embebida: la tapa mide 54 mm, así que
	// 640 px son ~300 ppp, nítida impresa sin inflar el PDF.
	ladoPortadaPx = 640
	// Mismo límite que la subida del logo: nada más grande puede estar en
	// el storage como portada de un proyecto.
	maximoBytesPortada  = TamanoMaximoLogoProyecto
	tiempoMaximoPortada = 5 * time.Second
	calidadJPEGPortada  = 88
)

// portadaPDF es la portada real del proyecto lista para el PDF.
type portadaPDF struct {
	// jpeg es la imagen recortada al cuadrado y escalada.
	jpeg []byte
	// colores son el fondo pastel y el acento derivados de la imagen, con el
	// mismo algoritmo que el Inicio (reporte_color.go).
	colores fondoPortada
}

// portadaParaPDF baja el logo del proyecto y lo deja listo para el PDF. Si
// algo falla devuelve nil y el PDF usa la portada generada: una portada que
// no se pudo leer no puede impedir exportar el reporte.
func portadaParaPDF(almacenamiento storage.AudioStorage, logo *string, codigoProyecto int64) *portadaPDF {
	if almacenamiento == nil || logo == nil || *logo == "" {
		return nil
	}

	ctx, cancelar := context.WithTimeout(context.Background(), tiempoMaximoPortada)
	defer cancelar()

	contenido, err := almacenamiento.Descargar(ctx, *logo, maximoBytesPortada)
	if err != nil {
		log.Printf("Reporte PDF: no se pudo descargar la portada del proyecto %d: %v", codigoProyecto, err)
		return nil
	}

	portada, err := prepararPortada(contenido)
	if err != nil {
		log.Printf("Reporte PDF: no se pudo procesar la portada del proyecto %d: %v", codigoProyecto, err)
		return nil
	}
	return portada
}

// prepararPortada recorta la imagen al cuadrado central (como object-cover
// en la app), la escala a ladoPortadaPx y la reencoda como JPEG: fpdf no lee
// WebP, y así el PDF no carga una foto de varios MB. Los colores se sacan de
// la imagen original completa, como en el Inicio.
func prepararPortada(contenido []byte) (*portadaPDF, error) {
	origen, _, err := image.Decode(bytes.NewReader(contenido))
	if err != nil {
		return nil, err
	}

	colores, ok := fondoDeImagen(origen)
	if !ok {
		// Imagen totalmente transparente: el mismo lavanda que usa el
		// Inicio para los proyectos sin portada.
		colores = fondoDesdeColor(colorHex("#8a7db8"))
	}

	limites := origen.Bounds()
	lado := min(limites.Dx(), limites.Dy())
	recorte := image.Rect(0, 0, lado, lado).Add(image.Point{
		X: limites.Min.X + (limites.Dx()-lado)/2,
		Y: limites.Min.Y + (limites.Dy()-lado)/2,
	})

	destino := image.NewRGBA(image.Rect(0, 0, ladoPortadaPx, ladoPortadaPx))
	// Fondo blanco: un PNG con transparencia no queda negro al pasar a JPEG.
	xdraw.Draw(destino, destino.Bounds(), image.NewUniform(color.White), image.Point{}, xdraw.Src)
	xdraw.CatmullRom.Scale(destino, destino.Bounds(), origen, recorte, xdraw.Over, nil)

	var salida bytes.Buffer
	if err := jpeg.Encode(&salida, destino, &jpeg.Options{Quality: calidadJPEGPortada}); err != nil {
		return nil, err
	}
	return &portadaPDF{jpeg: salida.Bytes(), colores: colores}, nil
}
