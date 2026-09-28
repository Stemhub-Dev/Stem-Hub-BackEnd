package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Parámetros compartidos por los listados paginados y filtrables
// (GET /canciones, GET /proyectos).

const (
	PaginaPorDefecto       = 1
	TamanoPaginaPorDefecto = 10
	TamanoPaginaMaximo     = 100

	// Tope defensivo para los filtros por código que viajan repetidos en la
	// query string (?proyectoId=1&proyectoId=2): no tiene sentido pedir más
	// valores que los que existen en un catálogo o que proyectos tiene un
	// usuario.
	MaximoCodigosFiltro = 100
)

// parsearEnteroPositivo lee un query param entero >= 1; si no viene, usa
// valorPorDefecto. Devuelve false si el valor es inválido.
func parsearEnteroPositivo(
	c *gin.Context,
	nombre string,
	valorPorDefecto int,
) (int, bool) {

	texto := strings.TrimSpace(c.Query(nombre))

	if texto == "" {
		return valorPorDefecto, true
	}

	valor, err := strconv.Atoi(texto)

	if err != nil || valor < 1 {
		return 0, false
	}

	return valor, true
}

// parsearCodigos lee un query param repetible de códigos (enteros >= 1).
// Sin valores devuelve nil, que los repositorios interpretan como "sin
// filtro". Devuelve un mensaje de error si algún valor es inválido.
func parsearCodigos(
	c *gin.Context,
	nombre string,
) ([]int64, string) {

	valores := c.QueryArray(nombre)

	if len(valores) == 0 {
		return nil, ""
	}

	mensajeError := nombre + " debe ser un entero mayor o igual a 1, hasta " +
		strconv.Itoa(MaximoCodigosFiltro) + " valores"

	if len(valores) > MaximoCodigosFiltro {
		return nil, mensajeError
	}

	codigos := make([]int64, 0, len(valores))

	for _, valor := range valores {

		codigo, err := strconv.ParseInt(
			strings.TrimSpace(valor),
			10,
			64,
		)

		if err != nil || codigo < 1 {
			return nil, mensajeError
		}

		codigos = append(codigos, codigo)
	}

	return codigos, ""
}

// parsearPaginacion lee page y pageSize con sus valores por defecto.
// Devuelve un mensaje de error si alguno es inválido.
func parsearPaginacion(
	c *gin.Context,
) (pagina int, tamanoPagina int, mensajeError string) {

	pagina, ok := parsearEnteroPositivo(
		c,
		"page",
		PaginaPorDefecto,
	)

	if !ok {
		return 0, 0, "page debe ser un entero mayor o igual a 1"
	}

	tamanoPagina, ok = parsearEnteroPositivo(
		c,
		"pageSize",
		TamanoPaginaPorDefecto,
	)

	if !ok || tamanoPagina > TamanoPaginaMaximo {
		return 0, 0, "pageSize debe ser un entero entre 1 y " +
			strconv.Itoa(TamanoPaginaMaximo)
	}

	return pagina, tamanoPagina, ""
}
