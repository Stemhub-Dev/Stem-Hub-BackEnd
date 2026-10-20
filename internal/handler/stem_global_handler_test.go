package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// HU-BUQ-B03: parámetros de GET /stems.

func filtroDe(t *testing.T, query string) (int64, string, int, int, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/stems"+query, nil)

	filtro, mensaje := parsearFiltroStemsGlobales(c)
	var categoria int64
	if filtro.Categoria != nil {
		categoria = *filtro.Categoria
	}
	return categoria, filtro.Busqueda, filtro.Pagina, filtro.TamanoPagina, mensaje
}

func TestFiltroStemsGlobales_CategoriaInvalidaEs400(t *testing.T) {
	for _, valor := range []string{"0", "-1", "abc", "1.5"} {
		t.Run(valor, func(t *testing.T) {
			if _, _, _, _, mensaje := filtroDe(t, "?categoriaId="+valor); mensaje != "categoriaId debe ser un entero mayor o igual a 1" {
				t.Fatalf("mensaje = %q", mensaje)
			}
		})
	}
}

func TestFiltroStemsGlobales_LeeCriterios(t *testing.T) {
	categoria, q, pagina, tamano, mensaje := filtroDe(t, "?q=%20bajo%20&categoriaId=2&page=3&pageSize=5")

	if mensaje != "" || categoria != 2 || q != "bajo" || pagina != 3 || tamano != 5 {
		t.Fatalf("categoria=%d q=%q page=%d pageSize=%d mensaje=%q", categoria, q, pagina, tamano, mensaje)
	}
}

func TestFiltroStemsGlobales_SinParametros(t *testing.T) {
	categoria, q, pagina, tamano, mensaje := filtroDe(t, "")

	if mensaje != "" || categoria != 0 || q != "" || pagina != 1 || tamano < 1 {
		t.Fatalf("categoria=%d q=%q page=%d pageSize=%d mensaje=%q", categoria, q, pagina, tamano, mensaje)
	}
}

func TestFiltroStemsGlobales_PaginacionInvalida(t *testing.T) {
	for _, query := range []string{"?page=0", "?pageSize=0", "?pageSize=1000"} {
		t.Run(query, func(t *testing.T) {
			if _, _, _, _, mensaje := filtroDe(t, query); mensaje == "" {
				t.Fatal("quería un error de paginación")
			}
		})
	}
}
