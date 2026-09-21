package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type EstadoProyectoHandler struct {
	service *service.EstadoProyectoService
}

func NewEstadoProyectoHandler(
	service *service.EstadoProyectoService,
) *EstadoProyectoHandler {

	return &EstadoProyectoHandler{
		service: service,
	}
}

func (h *EstadoProyectoHandler) Listar(
	c *gin.Context,
) {

	estados, err := h.service.Listar()

	if err != nil {

		log.Println(
			"Error al obtener estados de proyecto:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al obtener los estados de proyecto",
			},
		)

		return
	}

	c.JSON(
		http.StatusOK,
		estados,
	)
}
