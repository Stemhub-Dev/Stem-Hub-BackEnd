package dto

import "strconv"

// EtiquetaVersion es cómo se muestra el número de una versión en toda la
// app, en las respuestas, el resumen con IA y el PDF de reportes: "v3". El
// frontend usa el mismo formato (stemhub-frontend/src/lib/version.ts).
func EtiquetaVersion(numero int64) string {
	return "v" + strconv.FormatInt(numero, 10)
}
