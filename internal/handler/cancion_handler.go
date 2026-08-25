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

type CancionHandler struct {
	service service.CancionService
}

func NewCancionHandler(
	service service.CancionService,
) *CancionHandler {

	return &CancionHandler{
		service: service,
	}
}

func (h *CancionHandler) Crear(c *gin.Context) {

	codigoProyecto, err :=
		strconv.ParseInt(
			c.Param("proyectoId"),
			10,
			64,
		)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Proyecto inválido"},
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

	var request dto.CrearCancionRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	cancion, err :=
		h.service.Crear(
			usuario.CodigoUsuario,
			codigoProyecto,
			request,
		)

	switch {

	case errors.Is(
		err,
		service.ErrCancionNombreObligatorio,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "El nombre de la canción es obligatorio",
			},
		)

	case errors.Is(
		err,
		service.ErrCancionPistaObligatoria,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Debés cargar una pista de audio",
			},
		)

	case errors.Is(
		err,
		service.ErrCancionProyectoNoEncontrado,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "El proyecto no existe"},
		)

	case errors.Is(
		err,
		service.ErrCancionSinPermiso,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "No tenés permiso para crear canciones en este proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrCancionNombreDuplicado,
	):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": "Ya existe una canción con ese nombre en este proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrCancionPerfilRequerido,
	):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": "Completá tu perfil en StemHub",
			},
		)

	case err != nil:
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al crear la canción"},
		)

	default:
		c.JSON(
			http.StatusCreated,
			cancion,
		)
	}
}

func (h *CancionHandler) CrearVersion(c *gin.Context) {

	codigoProyecto, err :=
		strconv.ParseInt(
			c.Param("proyectoId"),
			10,
			64,
		)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Proyecto inválido"},
		)
		return
	}

	codigoCancion, err :=
		strconv.ParseInt(
			c.Param("cancionId"),
			10,
			64,
		)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Canción inválida"},
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

	var request dto.CrearVersionCancionRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	version, err :=
		h.service.CrearVersion(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoCancion,
			request,
		)
	switch {

	case errors.Is(
		err,
		service.ErrVersionPistaObligatoria,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Debés cargar una pista de audio para la nueva versión",
			},
		)

	case errors.Is(
		err,
		service.ErrVersionCancionNoEncontrada,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": "La canción no existe en este proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrVersionSinPermiso,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "No tenés permiso para crear versiones",
			},
		)

	case err != nil:
		log.Println(
			"Error al crear versión:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al crear la versión"},
		)

	default:
		c.JSON(
			http.StatusCreated,
			version,
		)
	}
}
