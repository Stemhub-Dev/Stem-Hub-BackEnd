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

func (h *PermisoHandler) Listar(
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

	permisos, err :=
		h.service.Listar(
			incluirInactivos,
		)

	if err != nil {

		log.Println(
			"Error al obtener permisos:",
			err,
		)

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

func (h *PermisoHandler) Crear(
	c *gin.Context,
) {

	var request dto.PermisoRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Solicitud inválida",
			},
		)
		return
	}

	permiso, err :=
		h.service.Crear(
			request,
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			service.ErrNombrePermisoRequerido,
		),
			errors.Is(
				err,
				service.ErrClavePermisoRequerida,
			),
			errors.Is(
				err,
				service.ErrClavePermisoInvalida,
			),
			errors.Is(
				err,
				service.ErrAmbitoPermisoRequerido,
			),
			errors.Is(
				err,
				service.ErrAmbitoPermisoInvalido,
			):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrNombrePermisoDuplicado,
		),
			errors.Is(
				err,
				service.ErrClavePermisoDuplicada,
			):

			c.JSON(
				http.StatusConflict,
				gin.H{
					"error": err.Error(),
				},
			)

		default:

			log.Println(
				"Error al crear permiso:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "Error al crear el permiso",
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusCreated,
		permiso,
	)
}

func (h *PermisoHandler) Editar(
	c *gin.Context,
) {

	codigoPermiso, err :=
		strconv.ParseInt(
			c.Param("id"),
			10,
			64,
		)

	if err != nil ||
		codigoPermiso <= 0 {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "ID de permiso inválido",
			},
		)
		return
	}

	var request dto.PermisoRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Solicitud inválida",
			},
		)
		return
	}

	permiso, err :=
		h.service.Editar(
			codigoPermiso,
			request,
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			service.ErrNombrePermisoRequerido,
		),
			errors.Is(
				err,
				service.ErrClavePermisoRequerida,
			),
			errors.Is(
				err,
				service.ErrClavePermisoInvalida,
			),
			errors.Is(
				err,
				service.ErrAmbitoPermisoRequerido,
			),
			errors.Is(
				err,
				service.ErrAmbitoPermisoInvalido,
			):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrPermisoNoEncontrado,
		):

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrNombrePermisoDuplicado,
		),
			errors.Is(
				err,
				service.ErrClavePermisoDuplicada,
			),
			errors.Is(
				err,
				service.ErrAmbitoPermisoNoEditable,
			),
			errors.Is(
				err,
				service.ErrClavePermisoNoEditable,
			):

			c.JSON(
				http.StatusConflict,
				gin.H{
					"error": err.Error(),
				},
			)

		default:

			log.Println(
				"Error al editar permiso:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "Error al editar el permiso",
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusOK,
		permiso,
	)
}

func (h *PermisoHandler) CambiarEstado(
	c *gin.Context,
) {

	codigoPermiso, err :=
		strconv.ParseInt(
			c.Param("id"),
			10,
			64,
		)

	if err != nil ||
		codigoPermiso <= 0 {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "ID de permiso inválido",
			},
		)
		return
	}

	var request dto.CambiarEstadoPermisoRequest

	if err :=
		c.ShouldBindJSON(
			&request,
		); err != nil {

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Solicitud inválida",
			},
		)
		return
	}

	permiso, err :=
		h.service.CambiarEstado(
			codigoPermiso,
			request,
		)

	if err != nil {

		switch {

		case errors.Is(
			err,
			service.ErrEstadoPermisoRequerido,
		):

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrPermisoNoEncontrado,
		):

			c.JSON(
				http.StatusNotFound,
				gin.H{
					"error": err.Error(),
				},
			)

		case errors.Is(
			err,
			service.ErrPermisoEnUso,
		):

			c.JSON(
				http.StatusConflict,
				gin.H{
					"error": err.Error(),
				},
			)

		default:

			log.Println(
				"Error al cambiar estado del permiso:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "Error al cambiar el estado del permiso",
				},
			)
		}

		return
	}

	c.JSON(
		http.StatusOK,
		permiso,
	)
}
