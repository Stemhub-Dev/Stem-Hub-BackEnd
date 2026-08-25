package handler

import (
	"errors"
	"log"
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

func (h *IntegranteHandler) ObtenerPerfil(c *gin.Context) {

	valorUsuario, existe :=
		c.Get(middleware.UsuarioContextKey)

	if !existe {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "Usuario no autenticado"},
		)
		return
	}

	usuario, ok :=
		valorUsuario.(*model.Usuario)

	if !ok || usuario == nil {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "Usuario no autenticado"},
		)
		return
	}

	integrante, err :=
		h.service.ObtenerPerfil(
			usuario.CodigoUsuario,
		)

	switch {

	case errors.Is(
		err,
		service.ErrPerfilNoEncontrado,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "Perfil no encontrado"},
		)

	case err != nil:
		log.Println(
			"Error al obtener perfil:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al obtener el perfil"},
		)

	default:
		c.JSON(
			http.StatusOK,
			dto.ObtenerPerfilResponse{
				CodigoIntegrante: integrante.CodIntegrante,
				Email:            usuario.Email,
				Nombre:           integrante.NombreIntegrante,
				Descripcion:      integrante.DescripcionIntegrante,
			},
		)
	}
}
