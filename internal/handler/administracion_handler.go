package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type UsuarioAdministracionHandler struct {
	service service.UsuarioAdministracionService
}

func NewUsuarioAdministracionHandler(
	service service.UsuarioAdministracionService,
) *UsuarioAdministracionHandler {
	return &UsuarioAdministracionHandler{
		service: service,
	}
}

func (h *UsuarioAdministracionHandler) Listar(c *gin.Context) {
	usuarios, err := h.service.Listar()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "no se pudieron obtener los usuarios",
		})
		return
	}

	c.JSON(http.StatusOK, usuarios)
}

func (h *UsuarioAdministracionHandler) ObtenerPorID(c *gin.Context) {
	codigoUsuario, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || codigoUsuario <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id de usuario inválido",
		})
		return
	}

	usuario, err := h.service.ObtenerPorID(codigoUsuario)
	if err != nil {
		if errors.Is(err, service.ErrUsuarioAdministracionNoEncontrado) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "usuario no encontrado",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "no se pudo obtener el usuario",
		})
		return
	}

	c.JSON(http.StatusOK, usuario)
}

func (h *UsuarioAdministracionHandler) ObtenerRolesSistema(c *gin.Context) {
	codigoUsuario, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || codigoUsuario <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id de usuario inválido",
		})
		return
	}

	response, err := h.service.ObtenerRolesSistema(codigoUsuario)
	if err != nil {
		if errors.Is(err, service.ErrUsuarioAdministracionNoEncontrado) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "usuario no encontrado",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "no se pudieron obtener los roles del usuario",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *UsuarioAdministracionHandler) ObtenerProyectosPorUsuario(c *gin.Context) {
	codigoUsuario, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || codigoUsuario <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id de usuario inválido",
		})
		return
	}

	response, err := h.service.ObtenerProyectosPorUsuario(codigoUsuario)
	if err != nil {
		if errors.Is(err, service.ErrUsuarioAdministracionNoEncontrado) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "usuario no encontrado",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "no se pudieron obtener los proyectos del usuario",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
