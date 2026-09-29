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

type TipoProyectoHandler struct {
	service *service.TipoProyectoService
}

func NewTipoProyectoHandler(
	service *service.TipoProyectoService,
) *TipoProyectoHandler {

	return &TipoProyectoHandler{
		service: service,
	}
}

func (h *TipoProyectoHandler) Listar(
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

	tipos, err :=
		h.service.Listar(
			incluirInactivos,
		)

	if err != nil {

		log.Println(
			"Error al obtener tipos de proyecto:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al obtener los tipos de proyecto",
			},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		tipos,
	)
}

func (h *TipoProyectoHandler) Crear(
	c *gin.Context,
) {

	var request dto.TipoProyectoRequest

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

	tipo, err :=
		h.service.Crear(
			request,
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			service.ErrNombreTipoProyectoRequerido,
		):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrTipoProyectoDuplicado,
		):

			c.JSON(
				http.StatusConflict,
				gin.H{
					"error": err.Error(),
				},
			)

		default:

			log.Println(
				"Error al crear tipo de proyecto:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "Error al crear el tipo de proyecto",
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusCreated,
		tipo,
	)
}

func (h *TipoProyectoHandler) Editar(
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
				"error": "ID de tipo de proyecto inválido",
			},
		)
		return
	}

	var request dto.TipoProyectoRequest

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

	tipo, err :=
		h.service.Editar(
			id,
			request,
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			service.ErrNombreTipoProyectoRequerido,
		):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrTipoProyectoNoEncontrado,
		):

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrTipoProyectoDuplicado,
		):

			c.JSON(
				http.StatusConflict,
				gin.H{
					"error": err.Error(),
				},
			)

		default:

			log.Println(
				"Error al editar tipo de proyecto:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "Error al editar el tipo de proyecto",
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusOK,
		tipo,
	)
}

func (h *TipoProyectoHandler) CambiarEstado(
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
				"error": "ID de tipo de proyecto inválido",
			},
		)
		return
	}

	var request dto.CambiarEstadoTipoProyectoRequest

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

	tipo, err :=
		h.service.CambiarEstado(
			id,
			request,
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			service.ErrEstadoTipoProyectoRequerido,
		):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrTipoProyectoNoEncontrado,
		):

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": err.Error(),
				},
			)

		default:

			log.Println(
				"Error al cambiar estado del tipo de proyecto:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "Error al cambiar el estado del tipo de proyecto",
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusOK,
		tipo,
	)
}
