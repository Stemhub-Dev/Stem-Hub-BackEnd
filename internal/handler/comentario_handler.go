package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type ComentarioHandler struct {
	service service.ComentarioService
}

func NewComentarioHandler(
	service service.ComentarioService,
) *ComentarioHandler {

	return &ComentarioHandler{
		service: service,
	}
}

func (h *ComentarioHandler) Crear(c *gin.Context) {

	codigoProyecto, err :=
		strconv.ParseInt(c.Param("proyectoId"), 10, 64)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Proyecto inválido"},
		)
		return
	}

	codigoCancion, err :=
		strconv.ParseInt(c.Param("cancionId"), 10, 64)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Canción inválida"},
		)
		return
	}

	codigoVersion, err :=
		strconv.ParseInt(c.Param("versionId"), 10, 64)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Versión inválida"},
		)
		return
	}

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

	var request dto.CrearComentarioRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	comentario, err :=
		h.service.Crear(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoCancion,
			codigoVersion,
			request,
		)

	switch {

	case errors.Is(
		err,
		service.ErrComentarioTextoObligatorio,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El comentario no puede estar vacío"},
		)

	case errors.Is(
		err,
		service.ErrComentarioTextoMuyLargo,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El comentario no puede superar los 200 caracteres"},
		)

	case errors.Is(
		err,
		service.ErrComentarioProyectoNoEncontrado,
	),
		errors.Is(
			err,
			service.ErrComentarioCancionNoEncontrada,
		),
		errors.Is(
			err,
			service.ErrComentarioVersionNoEncontrada,
		):

		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "El contenido solicitado no existe"},
		)

	case errors.Is(
		err,
		service.ErrComentarioSinAcceso,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "No tenés acceso a este proyecto"},
		)

	case err != nil:
		log.Println(
			"Error al crear comentario:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al crear el comentario"},
		)

	default:
		c.JSON(
			http.StatusCreated,
			comentario,
		)
	}
}
