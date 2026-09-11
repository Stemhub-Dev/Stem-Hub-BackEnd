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

type ProyectoHandler struct {
	service service.ProyectoService
}

func NewProyectoHandler(
	service service.ProyectoService,
) *ProyectoHandler {

	return &ProyectoHandler{
		service: service,
	}
}

func (h *ProyectoHandler) Crear(c *gin.Context) {

	usuarioContexto, existe :=
		c.Get(middleware.UsuarioContextKey)

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

	var request dto.CrearProyectoRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	proyecto, err := h.service.CrearProyecto(
		usuario.CodigoUsuario,
		request,
	)

	switch {

	case errors.Is(
		err,
		service.ErrNombreProyectoObligatorio,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El nombre del proyecto es obligatorio"},
		)

	case errors.Is(
		err,
		service.ErrPerfilRequerido,
	):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": "Completá tu perfil en StemHub antes de crear un proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrTipoProyectoNoValido,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El tipo de proyecto no es válido"},
		)

	case errors.Is(
		err,
		service.ErrRolProyectoNoValido,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El rol seleccionado no es válido para un proyecto"},
		)

	case errors.Is(
		err,
		service.ErrGenerosNoValidos,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Uno o más géneros no son válidos"},
		)

	case err != nil:
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al crear el proyecto"},
		)

	default:
		c.JSON(
			http.StatusCreated,
			proyecto,
		)
	}
}

func (h *ProyectoHandler) Listar(c *gin.Context) {

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

	proyectos, err :=
		h.service.ListarProyectos(
			usuario.CodigoUsuario,
		)

	if err != nil {

		log.Println(
			"Error al listar proyectos:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al obtener los proyectos",
			},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		proyectos,
	)
}

func (h *ProyectoHandler) ListarColaboradores(c *gin.Context) {

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

	codigoProyecto, err :=
		strconv.ParseInt(c.Param("proyectoId"), 10, 64)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Proyecto inválido"},
		)
		return
	}

	colaboradores, err :=
		h.service.ListarColaboradores(
			usuario.CodigoUsuario,
			codigoProyecto,
		)

	switch {

	case errors.Is(
		err,
		service.ErrProyectoNoEncontrado,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "El proyecto no existe"},
		)

	case errors.Is(
		err,
		service.ErrProyectoSinAcceso,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "No tenés acceso a este proyecto"},
		)

	case err != nil:
		log.Println(
			"Error al listar colaboradores:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al obtener los colaboradores"},
		)

	default:
		c.JSON(
			http.StatusOK,
			colaboradores,
		)
	}
}
