package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// pngDePrueba arma un PNG apaisado con transparencia: obliga a recortar al
// cuadrado y a aplanar el canal alfa.
func pngDePrueba(t *testing.T, ancho, alto int) []byte {
	t.Helper()
	imagen := image.NewNRGBA(image.Rect(0, 0, ancho, alto))
	for x := 0; x < ancho; x++ {
		for y := 0; y < alto; y++ {
			imagen.Set(x, y, color.NRGBA{R: uint8(x), G: 98, B: 160, A: uint8(y)})
		}
	}
	var salida bytes.Buffer
	if err := png.Encode(&salida, imagen); err != nil {
		t.Fatal(err)
	}
	return salida.Bytes()
}

func TestPrepararPortada_RecortaAlCuadradoYDevuelveJPEG(t *testing.T) {
	portada, err := prepararPortada(pngDePrueba(t, 300, 200))
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	imagen, err := jpeg.Decode(bytes.NewReader(portada.jpeg))
	if err != nil {
		t.Fatalf("la portada no es un JPEG: %v", err)
	}
	if limites := imagen.Bounds(); limites.Dx() != ladoPortadaPx || limites.Dy() != ladoPortadaPx {
		t.Errorf("tamaño = %dx%d, quería %dx%d", limites.Dx(), limites.Dy(), ladoPortadaPx, ladoPortadaPx)
	}
}

// Los colores salen del tono de la imagen: una portada toda roja da un fondo
// pastel rojizo y un acento rojo oscuro.
func TestPrepararPortada_DevuelveLosColoresDeLaImagen(t *testing.T) {
	imagen := image.NewNRGBA(image.Rect(0, 0, 50, 50))
	for x := 0; x < 50; x++ {
		for y := 0; y < 50; y++ {
			imagen.Set(x, y, color.NRGBA{R: 200, G: 40, B: 40, A: 255})
		}
	}
	var contenido bytes.Buffer
	if err := png.Encode(&contenido, imagen); err != nil {
		t.Fatal(err)
	}

	portada, err := prepararPortada(contenido.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if esperado := fondoDesdeColor(colorRGB{200, 40, 40}); portada.colores != esperado {
		t.Errorf("colores = %+v, quería %+v", portada.colores, esperado)
	}
}

func TestPrepararPortada_RechazaLoQueNoEsImagen(t *testing.T) {
	if _, err := prepararPortada([]byte("no soy una imagen")); err == nil {
		t.Error("se esperaba un error")
	}
}

// storagePortadaFalso devuelve un contenido fijo y registra qué se pidió.
type storagePortadaFalso struct {
	storageFalso
	contenido []byte
	err       error
	pedidos   []string
}

func (s *storagePortadaFalso) Descargar(_ context.Context, key string, _ int64) ([]byte, error) {
	s.pedidos = append(s.pedidos, key)
	return s.contenido, s.err
}

const logoDePrueba = "proyectos/3/logo.png"

func TestPortadaParaPDF_DescargaYPreparaElLogo(t *testing.T) {
	logo := logoDePrueba
	almacenamiento := &storagePortadaFalso{contenido: pngDePrueba(t, 120, 120)}

	portada := portadaParaPDF(almacenamiento, &logo, 3)
	if portada == nil || len(portada.jpeg) == 0 {
		t.Fatal("se esperaba una portada")
	}
	if len(almacenamiento.pedidos) != 1 || almacenamiento.pedidos[0] != logo {
		t.Errorf("pedidos = %v", almacenamiento.pedidos)
	}
}

func TestPortadaParaPDF_SinLogoNoConsultaElStorage(t *testing.T) {
	vacio := ""
	almacenamiento := &storagePortadaFalso{}
	for _, clave := range []*string{nil, &vacio} {
		if portadaParaPDF(almacenamiento, clave, 3) != nil {
			t.Error("sin logo se esperaba nil")
		}
	}
	if len(almacenamiento.pedidos) != 0 {
		t.Errorf("no debería descargar nada: %v", almacenamiento.pedidos)
	}
}

// En todos estos casos el PDF tiene que caer en la portada generada.
func TestPortadaParaPDF_UsaLaGeneradaSiNoPuedeLeerla(t *testing.T) {
	logo := logoDePrueba
	casos := map[string]*storagePortadaFalso{
		"falla la descarga": {err: errors.New("minio caído")},
		"no es una imagen":  {contenido: []byte("texto")},
	}
	for nombre, almacenamiento := range casos {
		if portadaParaPDF(almacenamiento, &logo, 3) != nil {
			t.Errorf("%s: se esperaba nil", nombre)
		}
	}
	if portadaParaPDF(nil, &logo, 3) != nil {
		t.Error("sin storage: se esperaba nil")
	}
}

func TestConstruirReportePDF_ConPortadaReal(t *testing.T) {
	portada, err := prepararPortada(pngDePrueba(t, 200, 200))
	if err != nil {
		t.Fatal(err)
	}

	contenido, err := construirReportePDF(reporteDePrueba(ReporteEstadoCanciones, []map[string]interface{}{
		filaEstado("Balada", 3, 2, 1),
	}), portada)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !bytes.Contains(contenido, []byte("/Subtype /Image")) {
		t.Error("el PDF no tiene la imagen de la portada")
	}
}

// Una portada corrupta no puede romper la exportación: se reintenta sin ella.
func TestConstruirReportePDF_PortadaInvalidaUsaLaGenerada(t *testing.T) {
	contenido, err := construirReportePDF(reporteDePrueba(ReporteEstadoCanciones, nil), &portadaPDF{jpeg: []byte("no es un jpeg")})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !bytes.HasPrefix(contenido, []byte("%PDF-")) || bytes.Contains(contenido, []byte("/Subtype /Image")) {
		t.Error("se esperaba un PDF sin imagen")
	}
}
