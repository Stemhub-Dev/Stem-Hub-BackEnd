package service

import "testing"

// Mismos casos que stemhub-frontend/src/lib/colorPortada.test.ts: el PDF y
// el Inicio tienen que dar el mismo color para la misma portada.

func pixelesRGBA(colores ...[4]uint8) []uint8 {
	salida := []uint8{}
	for _, c := range colores {
		salida = append(salida, c[:]...)
	}
	return salida
}

func TestColorDominante(t *testing.T) {
	t.Run("le da más peso a los colores intensos que a los grises", func(t *testing.T) {
		gris := [4]uint8{200, 200, 200, 255}
		color, ok := colorDominante(pixelesRGBA(gris, gris, gris, gris, [4]uint8{240, 80, 160, 255}))
		if !ok || color.r-color.g <= 80 {
			t.Errorf("color = %+v", color)
		}
	})

	t.Run("gana el tono que más área ocupa aunque haya detalles más saturados", func(t *testing.T) {
		rosa := [4]uint8{246, 182, 208, 255}
		rojo := [4]uint8{220, 30, 40, 255}
		amarillo := [4]uint8{250, 200, 40, 255}
		color, _ := colorDominante(pixelesRGBA(rosa, rosa, rosa, rosa, rosa, rosa, rosa, rojo, rojo, amarillo, amarillo))
		if color != (colorRGB{246, 182, 208}) {
			t.Errorf("color = %+v, quería el rosa", color)
		}
	})

	t.Run("una portada sin color promedia todo", func(t *testing.T) {
		color, _ := colorDominante(pixelesRGBA([4]uint8{10, 10, 10, 255}, [4]uint8{30, 30, 30, 255}))
		if color != (colorRGB{20, 20, 20}) {
			t.Errorf("color = %+v", color)
		}
	})

	t.Run("ignora los píxeles transparentes", func(t *testing.T) {
		color, _ := colorDominante(pixelesRGBA([4]uint8{255, 0, 0, 0}, [4]uint8{0, 0, 255, 255}))
		if color != (colorRGB{0, 0, 255}) {
			t.Errorf("color = %+v", color)
		}
		if _, ok := colorDominante(pixelesRGBA([4]uint8{255, 0, 0, 10})); ok {
			t.Error("sin píxeles visibles no hay color")
		}
	})
}

func TestFondoDesdeColor(t *testing.T) {
	brillo := func(c colorRGB) int { return max(c.r, c.g, c.b) }

	casos := map[string]colorRGB{
		"rosa claro":  {246, 182, 208},
		"amarillo":    {247, 226, 122},
		"azul oscuro": {20, 30, 60},
		"blanco":      {255, 255, 255},
		"negro":       {0, 0, 0},
	}
	for nombre, color := range casos {
		fondo := fondoDesdeColor(color)
		if brillo(fondo.fondo) <= 225 {
			t.Errorf("%s: el fondo %+v no es claro", nombre, fondo.fondo)
		}
		if contrasteEntre(fondo.acento, fondo.fondo) < 4.5 {
			t.Errorf("%s: el acento no se lee sobre el fondo", nombre)
		}
		if contrasteEntre(colorInk900, fondo.fondo) < 4.5 {
			t.Errorf("%s: la tinta no se lee sobre el fondo", nombre)
		}
		// La píldora del estado es texto blanco sobre el acento.
		if contrasteEntre(colorBlanco, fondo.acento) < 4.5 {
			t.Errorf("%s: el texto blanco no se lee sobre el acento", nombre)
		}
	}

	t.Run("mantiene el tono de la portada", func(t *testing.T) {
		fondo := fondoDesdeColor(colorRGB{246, 182, 208})
		for _, c := range []colorRGB{fondo.fondo, fondo.acento} {
			if c.r <= c.b || c.b <= c.g {
				t.Errorf("%+v ya no es rosa", c)
			}
		}
	})
}

// Cada portada generada da colores que se leen bien en la banda.
func TestFondoDeLasPortadasGeneradas(t *testing.T) {
	for indice, diseno := range disenosPortada {
		fondo := fondoDesdeColor(diseno.fondo)
		if contrasteEntre(colorInk900, fondo.fondo) < 4.5 || contrasteEntre(fondo.acento, fondo.fondo) < 4.5 {
			t.Errorf("diseño %d: contraste insuficiente", indice)
		}
	}
}
