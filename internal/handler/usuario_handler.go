package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type UsuarioHandler struct {
	service service.UsuarioService
}

func NewUsuarioHandler(service service.UsuarioService) *UsuarioHandler {
	return &UsuarioHandler{
		service: service,
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
