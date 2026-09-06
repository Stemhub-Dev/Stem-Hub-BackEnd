package handler

import (
	"net/http"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type PermisoHandler struct {
	service service.PermisoService
}

func NewPermisoHandler(
	service service.PermisoService,
) *PermisoHandler {

	return &PermisoHandler{
		service: service,
	}
}

func (h *PermisoHandler) Listar(c *gin.Context) {

	permisos, err :=
		h.service.ListarActivos()

	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al obtener los permisos",
			},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		permisos,
	)
}
