package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type ManualHandler struct {
	service *service.ManualService
}

func NewManualHandler(service *service.ManualService) *ManualHandler {
	return &ManualHandler{
		service: service,
	}
}

func (h *ManualHandler) DescargarManual(c *gin.Context) {
	pdfBytes, err := h.service.ObtenerManualUsuario(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "manual de usuario no encontrado",
		})
		return
	}

	c.Header("Content-Disposition", `attachment; filename="manual-usuario.pdf"`)
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
