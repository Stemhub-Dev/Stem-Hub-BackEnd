package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
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

	incluirInactivos := false

	if valor, existe :=
		c.GetQuery("incluirInactivos"); existe {

		parsed, err :=
			strconv.ParseBool(valor)

		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": "El parámetro incluirInactivos es inválido",
				},
			)
			return
		}

		incluirInactivos = parsed
	}

	estados, err :=
		h.service.Listar(
			incluirInactivos,
		)

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

func (h *EstadoProyectoHandler) Crear(
	c *gin.Context,
) {

	var request dto.EstadoProyectoRequest

	if err := c.ShouldBindJSON(&request); err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Solicitud inválida",
			},
		)
		return
	}

	estado, err :=
		h.service.Crear(
			request,
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			service.ErrNombreEstadoProyectoRequerido,
		):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrEstadoProyectoDuplicado,
		):

			c.JSON(
				http.StatusConflict,
				gin.H{
					"error": err.Error(),
				},
			)

		default:

			log.Println(
				"Error al crear estado de proyecto:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "Error al crear el estado de proyecto",
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusCreated,
		estado,
	)
}

func (h *EstadoProyectoHandler) Editar(
	c *gin.Context,
) {

	id, err :=
		strconv.ParseInt(
			c.Param("id"),
			10,
			64,
		)

	if err != nil || id <= 0 {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "ID de estado de proyecto inválido",
			},
		)
		return
	}

	var request dto.EstadoProyectoRequest

	if err :=
		c.ShouldBindJSON(&request); err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Solicitud inválida",
			},
		)
		return
	}

	estado, err :=
		h.service.Editar(
			id,
			request,
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			service.ErrNombreEstadoProyectoRequerido,
		):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrEstadoProyectoNoEncontrado,
		):

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrEstadoProyectoDuplicado,
		):

			c.JSON(
				http.StatusConflict,
				gin.H{
					"error": err.Error(),
				},
			)

		default:

			log.Println(
				"Error al editar estado de proyecto:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "Error al editar el estado de proyecto",
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusOK,
		estado,
	)
}

func (h *EstadoProyectoHandler) CambiarEstado(
	c *gin.Context,
) {

	id, err :=
		strconv.ParseInt(
			c.Param("id"),
			10,
			64,
		)

	if err != nil || id <= 0 {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "ID de estado de proyecto inválido",
			},
		)
		return
	}

	var request dto.CambiarEstadoProyectoRequest

	if err :=
		c.ShouldBindJSON(&request); err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Solicitud inválida",
			},
		)
		return
	}

	estado, err :=
		h.service.CambiarEstado(
			id,
			request,
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			service.ErrEstadoEstadoProyectoRequerido,
		):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrEstadoProyectoNoEncontrado,
		):

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": err.Error(),
				},
			)

		default:

			log.Println(
				"Error al cambiar estado de proyecto:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "Error al cambiar el estado del estado de proyecto",
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusOK,
		estado,
	)
}
