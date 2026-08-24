package handler

import (
	"errors"
	"net/http"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type IntegranteHandler struct {
	service service.IntegranteService
}

func NewIntegranteHandler(
	service service.IntegranteService,
) *IntegranteHandler {
	return &IntegranteHandler{
		service: service,
	}
}

func (h *IntegranteHandler) CrearPerfil(c *gin.Context) {

	usuarioContexto, existe := c.Get(
		middleware.UsuarioContextKey,
	)

	if !existe {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "Usuario no autenticado"},
		)
		return
	}

	usuario, ok := usuarioContexto.(*model.Usuario)

	if !ok || usuario == nil {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "Usuario no autenticado"},
		)
		return
	}

	var request dto.CrearPerfilIntegranteRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	integrante, err := h.service.CrearPerfil(
		usuario.CodigoUsuario,
		request.Nombre,
		request.Descripcion,
	)

	switch {
	case errors.Is(
		err,
		service.ErrNombreIntegranteObligatorio,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El nombre es obligatorio"},
		)

	case errors.Is(
		err,
		service.ErrPerfilIntegranteYaExiste,
	):
		c.JSON(
			http.StatusConflict,
			gin.H{"error": "El usuario ya posee un perfil en StemHub"},
		)

	case err != nil:
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al crear el perfil"},
		)

	default:
		c.JSON(
			http.StatusCreated,
			integrante,
		)
	}
}
