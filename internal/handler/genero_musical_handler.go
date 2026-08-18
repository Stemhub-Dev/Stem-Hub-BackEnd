package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type GeneroMusicalHandler struct {
	service *service.GeneroMusicalService
}

func NewGeneroMusicalHandler(service *service.GeneroMusicalService) *GeneroMusicalHandler {
	return &GeneroMusicalHandler{
		service: service,
	}
}

func (h *GeneroMusicalHandler) Listar(c *gin.Context) {
	generos, err := h.service.Listar()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener los géneros musicales",
		})
		return
	}

	c.JSON(http.StatusOK, generos)
}

func (h *GeneroMusicalHandler) Crear(c *gin.Context) {
	var request dto.GeneroMusicalRequest

	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Solicitud inválida",
		})
		return
	}

	genero, err := h.service.Crear(request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNombreGeneroMusicalRequerido):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, service.ErrGeneroMusicalDuplicado):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error al crear el género musical",
			})
		}

		return
	}

	c.JSON(http.StatusCreated, genero)
}

func (h *GeneroMusicalHandler) Editar(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID de género musical inválido",
		})
		return
	}

	var request dto.GeneroMusicalRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Solicitud inválida",
		})
		return
	}

	genero, err := h.service.Editar(id, request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNombreGeneroMusicalRequerido):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, service.ErrGeneroMusicalNoEncontrado):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, service.ErrGeneroMusicalDuplicado):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error al editar el género musical",
			})
		}

		return
	}

	c.JSON(http.StatusOK, genero)
}

func (h *GeneroMusicalHandler) CambiarEstado(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID de género musical inválido",
		})
		return
	}

	var request dto.CambiarEstadoGeneroMusicalRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Solicitud inválida",
		})
		return
	}

	genero, err := h.service.CambiarEstado(id, request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEstadoGeneroMusicalRequerido):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})

		case errors.Is(err, service.ErrGeneroMusicalNoEncontrado):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error al cambiar el estado del género musical",
			})
		}

		return
	}

	c.JSON(http.StatusOK, genero)
}
