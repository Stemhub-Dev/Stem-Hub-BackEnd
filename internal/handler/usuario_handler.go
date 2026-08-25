package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type UsuarioHandler struct {
	service           service.UsuarioService
	usuarioRolService service.UsuarioRolService
}

func NewUsuarioHandler(service service.UsuarioService, usuarioRolService service.UsuarioRolService) *UsuarioHandler {
	return &UsuarioHandler{
		service:           service,
		usuarioRolService: usuarioRolService,
	}
}

func (h *UsuarioHandler) Registrar(c *gin.Context) {
	claimsValue, existe := c.Get(middleware.ClaimsContextKey)

	if !existe {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "identidad de autenticación no disponible",
		})
		return
	}

	claims, ok := claimsValue.(*middleware.SupabaseClaims)

	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "error interno del servidor",
		})
		return
	}

	if claims.Email == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "el token no contiene email",
		})
		return
	}

	usuario, creado, err := h.service.RegistrarUsuario(
		claims.Subject,
		claims.Email,
	)

	if err != nil {
		if errors.Is(err, service.ErrUsuarioInactivo) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "usuario inactivo",
			})
			return
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "error interno del servidor",
		})
		return
	}

	if creado {
		c.JSON(http.StatusCreated, usuario)
		return
	}

	c.JSON(http.StatusOK, usuario)
}

func (h *UsuarioHandler) AsignarRol(c *gin.Context) {

	codigoUsuario, err := strconv.ParseInt(
		c.Param("codigoUsuario"),
		10,
		64,
	)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Código de usuario inválido"},
		)
		return
	}

	var request dto.AsignarRolUsuarioRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	err = h.usuarioRolService.AsignarRol(
		codigoUsuario,
		request.CodRol,
	)

	switch {
	case errors.Is(
		err,
		service.ErrUsuarioRolUsuarioNoEncontrado,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "El usuario no existe"},
		)

	case errors.Is(
		err,
		service.ErrUsuarioRolRolNoEncontrado,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "El rol no existe"},
		)

	case errors.Is(
		err,
		service.ErrUsuarioRolYaAsignado,
	):
		c.JSON(
			http.StatusConflict,
			gin.H{"error": "El usuario ya posee ese rol"},
		)

	case err != nil:
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al asignar el rol"},
		)

	default:
		c.JSON(
			http.StatusCreated,
			gin.H{"mensaje": "Rol asignado correctamente"},
		)
	}
}
