package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type TipoProyectoHandler struct {
	service *service.TipoProyectoService
}

func NewTipoProyectoHandler(service *service.TipoProyectoService) *TipoProyectoHandler {
	return &TipoProyectoHandler{
		service: service,
	}
}

func (h *TipoProyectoHandler) Listar(c *gin.Context) {
	tipos, err := h.service.Listar()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener los tipos de proyecto",
		})
		return
	}

	c.JSON(http.StatusOK, tipos)
}
