package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
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

		log.Printf(
			"Error al obtener permisos del rol %d: %v",
			codigoRol,
			err,
		)

		if errors.Is(
			err,
			service.ErrRolPermisoRolNoEncontrado,
		) {
			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": "rol no encontrado",
				},
			)
			return
		}

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "no se pudieron obtener los permisos del rol",
			},
		)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *RolPermisoHandler) Actualizar(
	c *gin.Context,
) {

	codigoRol, err :=
		strconv.ParseInt(
			c.Param("id"),
			10,
			64,
		)

	if err != nil ||
		codigoRol <= 0 {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "id de rol inválido",
			},
		)
		return
	}

	var request dto.ActualizarPermisosRolRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "solicitud inválida",
			},
		)
		return
	}

	response, err :=
		h.service.ActualizarPermisosRol(
			codigoRol,
			request,
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			service.ErrRolPermisoRolNoEncontrado,
		):

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrRolPermisoPermisosInvalidos,
		):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrRolPermisoAdministradorSinGestionRoles,
		):

			c.JSON(
				http.StatusConflict,
				gin.H{
					"error": err.Error(),
				},
			)

		default:

			log.Printf(
				"Error al actualizar permisos del rol %d: %v",
				codigoRol,
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "no se pudieron actualizar los permisos del rol",
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusOK,
		response,
	)
}
