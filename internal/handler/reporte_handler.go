package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type ReporteHandler struct{ service service.ReporteService }

func NewReporteHandler(reporteService service.ReporteService) *ReporteHandler {
	return &ReporteHandler{service: reporteService}
}

func (h *ReporteHandler) Listar(c *gin.Context) {
	usuario, ok := usuarioDesdeContexto(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return
	}
	proyecto, err := strconv.ParseInt(c.Query("proyectoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "proyectoId es obligatorio y debe ser válido"})
		return
	}
	reportes, err := h.service.ListarCatalogo(usuario.CodigoUsuario, proyecto)
	if h.responderError(c, err) {
		return
	}
	if len(reportes) == 0 {
		c.JSON(http.StatusNoContent, []dto.ReporteCatalogoResponse{})
		return
	}
	c.JSON(http.StatusOK, reportes)
}

func (h *ReporteHandler) Generar(c *gin.Context) {
	usuario, ok := usuarioDesdeContexto(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return
	}
	var request dto.GenerarReporteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Los parámetros del reporte son inválidos"})
		return
	}
	reporte, err := h.service.Generar(usuario.CodigoUsuario, c.Param("tipo"), request)
	if h.responderError(c, err) {
		return
	}
	if len(reporte.Datos) == 0 {
		c.JSON(http.StatusNoContent, reporte)
		return
	}
	c.JSON(http.StatusOK, reporte)
}

func (h *ReporteHandler) ExportarPDF(c *gin.Context) {
	usuario, ok := usuarioDesdeContexto(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return
	}
	var request dto.GenerarReporteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Los parámetros del reporte son inválidos"})
		return
	}
	pdf, nombre, err := h.service.ExportarPDF(usuario.CodigoUsuario, c.Param("tipo"), request)
	if h.responderError(c, err) {
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+nombre+`"`)
	c.Data(http.StatusOK, "application/pdf", pdf)
}

func usuarioDesdeContexto(c *gin.Context) (*model.Usuario, bool) {
	valor, existe := c.Get(middleware.UsuarioContextKey)
	usuario, ok := valor.(*model.Usuario)
	return usuario, existe && ok && usuario != nil
}

func (h *ReporteHandler) responderError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrReporteTipoNoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"error": "El reporte no existe"})
	case errors.Is(err, service.ErrReporteProyectoNoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"error": "El proyecto no existe"})
	case errors.Is(err, service.ErrReporteFiltrosInvalidos):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Los filtros del reporte son inválidos"})
	case errors.Is(err, service.ErrReporteSinPermiso):
		c.JSON(http.StatusForbidden, gin.H{"error": "No tenés permiso para consultar reportes de este proyecto"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo procesar el reporte"})
	}
	return true
}
