package service

import "unicode/utf8"

// Largos máximos de los textos que carga el usuario, en caracteres (runas,
// no bytes). Más cortos que las columnas: un nombre largo rompe las
// pantallas donde se muestra. El front usa los mismos valores
// (stemhub-frontend: src/lib/forms/largosMaximos.ts).
const (
	LargoMaximoNombreProyecto      = 60
	LargoMaximoDescripcionProyecto = 500
	LargoMaximoNombreCancion       = 80
	LargoMaximoNotasVersion        = 500
)

func excedeLargo(texto string, maximo int) bool {
	return utf8.RuneCountInString(texto) > maximo
}
