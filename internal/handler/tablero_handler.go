package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// TableroHandler expone los datos del Tablero (HU-DASH-B01/B02). Cada
// gráfico tiene su propio endpoint para que el frontend pueda cargarlo y
// reintentarlo sin depender del resto.
type TableroHandler struct {
	service service.TableroService
}

func NewTableroHandler(tableroService service.TableroService) *TableroHandler {
	return &TableroHandler{service: tableroService}
}

// GET /tablero/indicadores?proyectoId=
func (h *TableroHandler) ObtenerIndicadores(c *gin.Context) {
	codigoUsuario, codigoProyecto, ok := h.leerParametrosComunes(c)
	if !ok {
		return
	}

	indicadores, err := h.service.ObtenerIndicadores(codigoUsuario, codigoProyecto)
	if h.responderError(c, err) {
		return
	}

	c.JSON(http.StatusOK, indicadores)
}

// GET /tablero/graficos/versiones-por-proyecto?proyectoId=
func (h *TableroHandler) ObtenerVersionesPorProyecto(c *gin.Context) {
	codigoUsuario, codigoProyecto, ok := h.leerParametrosComunes(c)
	if !ok {
		return
	}

	datos, err := h.service.ObtenerVersionesPorProyecto(codigoUsuario, codigoProyecto)
	if h.responderError(c, err) {
		return
	}

	c.JSON(http.StatusOK, datos)
}

// GET /tablero/graficos/canciones-por-etapa?proyectoId=
func (h *TableroHandler) ObtenerCancionesPorEtapa(c *gin.Context) {
	codigoUsuario, codigoProyecto, ok := h.leerParametrosComunes(c)
	if !ok {
		return
	}

	datos, err := h.service.ObtenerCancionesPorEtapa(codigoUsuario, codigoProyecto)
	if h.responderError(c, err) {
		return
	}

	c.JSON(http.StatusOK, datos)
}

// GET /tablero/graficos/actividad?agrupacion=semanal|mensual&proyectoId=
func (h *TableroHandler) ObtenerActividad(c *gin.Context) {
	codigoUsuario, codigoProyecto, ok := h.leerParametrosComunes(c)
	if !ok {
		return
	}

	agrupacion := strings.TrimSpace(c.Query("agrupacion"))
	if agrupacion == "" {
		agrupacion = service.AgrupacionSemanal
	}

	datos, err := h.service.ObtenerActividad(codigoUsuario, codigoProyecto, agrupacion)
	if h.responderError(c, err) {
		return
	}

	c.JSON(http.StatusOK, datos)
}

// leerParametrosComunes obtiene el usuario autenticado y el filtro opcional
// proyectoId. Si algo falla ya respondió y devuelve false.
func (h *TableroHandler) leerParametrosComunes(c *gin.Context) (int64, *int64, bool) {
	usuario, ok := usuarioDesdeContexto(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return 0, nil, false
	}

	texto := strings.TrimSpace(c.Query("proyectoId"))
	if texto == "" {
		return usuario.CodigoUsuario, nil, true
	}

	codigoProyecto, err := strconv.ParseInt(texto, 10, 64)
	if err != nil || codigoProyecto < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proyectoId debe ser un número entero positivo"})
		return 0, nil, false
	}

	return usuario.CodigoUsuario, &codigoProyecto, true
}

func (h *TableroHandler) responderError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, service.ErrTableroAgrupacionInvalida):
		c.JSON(http.StatusBadRequest, gin.H{"error": "agrupacion debe ser semanal o mensual"})
	case errors.Is(err, service.ErrTableroProyectoNoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"error": "El proyecto no existe"})
	case errors.Is(err, service.ErrTableroSinAcceso):
		c.JSON(http.StatusForbidden, gin.H{"error": "No participás de este proyecto"})
	default:
		log.Printf("Error al obtener datos del Tablero (%s): %v", c.FullPath(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudieron obtener los datos del Tablero"})
	}

	return true
}
