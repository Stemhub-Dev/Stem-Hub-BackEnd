package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type AuditoriaHandler struct {
	service service.AuditoriaService
}

func NewAuditoriaHandler(
	auditoriaService service.AuditoriaService,
) *AuditoriaHandler {
	return &AuditoriaHandler{service: auditoriaService}
}

func (h *AuditoriaHandler) Listar(c *gin.Context) {
	pagina, tamanoPagina, mensaje := parsearPaginacion(c)
	if mensaje != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": mensaje})
		return
	}

	codigoUsuario, err := leerCodigoAuditoria(c, "usuarioId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	codigoProyecto, err := leerCodigoAuditoria(c, "proyectoId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	desde, err := leerFechaAuditoria(c, "desde")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hasta, err := leerFechaAuditoria(c, "hasta")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resultado, err := h.service.Listar(
		c.Request.Context(),
		dto.AuditoriaFiltro{
			Pagina:         pagina,
			TamanoPagina:   tamanoPagina,
			Entidad:        c.Query("entidad"),
			Evento:         c.Query("evento"),
			CodigoUsuario:  codigoUsuario,
			CodigoProyecto: codigoProyecto,
			Desde:          desde,
			Hasta:          hasta,
		},
	)

	if errors.Is(err, service.ErrFiltroAuditoriaInvalido) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Filtros inválidos: verificá los valores y que desde sea anterior a hasta",
		})
		return
	}

	if err != nil {
		log.Printf("Error al consultar auditoría: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "No se pudo consultar la auditoría",
		})
		return
	}

	c.JSON(http.StatusOK, resultado)
}

func leerCodigoAuditoria(
	c *gin.Context,
	nombre string,
) (*int64, error) {
	texto := strings.TrimSpace(c.Query(nombre))
	if texto == "" {
		return nil, nil
	}

	codigo, err := strconv.ParseInt(texto, 10, 64)
	if err != nil || codigo <= 0 {
		return nil, errors.New(nombre + " debe ser un entero positivo")
	}

	return &codigo, nil
}

func leerFechaAuditoria(
	c *gin.Context,
	nombre string,
) (*time.Time, error) {
	texto := strings.TrimSpace(c.Query(nombre))
	if texto == "" {
		return nil, nil
	}

	fecha, err := time.Parse(time.RFC3339, texto)
	if err != nil {
		return nil, errors.New(
			nombre + " debe tener formato RFC3339, por ejemplo 2026-11-08T00:00:00-03:00",
		)
	}

	return &fecha, nil
}
