package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type RolPermisoHandler struct {
	service service.RolPermisoService
}

func NewRolPermisoHandler(
	service service.RolPermisoService,
) *RolPermisoHandler {
	return &RolPermisoHandler{
		service: service,
	}
}

func (h *RolPermisoHandler) ObtenerPorRol(c *gin.Context) {
	codigoRol, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || codigoRol <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id de rol inválido",
		})
		return
	}

	response, err := h.service.ObtenerPermisosPorRol(codigoRol)
	if err != nil {
		if errors.Is(err, service.ErrRolPermisoRolNoEncontrado) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "rol no encontrado",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "no se pudieron obtener los permisos del rol",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
