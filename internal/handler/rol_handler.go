package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type RolHandler struct {
	service *service.RolService
}

func NewRolHandler(service *service.RolService) *RolHandler {
	return &RolHandler{
		service: service,
	}
}

func (h *RolHandler) Listar(c *gin.Context) {
	if c.Request.Method != http.MethodGet {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Método no permitido"})
		return
	}

	roles, err := h.service.Listar()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener los roles"})
		return
	}

	c.JSON(http.StatusOK, roles)
}
