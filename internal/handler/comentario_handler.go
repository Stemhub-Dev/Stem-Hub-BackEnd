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
		service.ErrComentarioRangoInvalido,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El rango de tiempo del comentario es inválido"},
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

func (h *ComentarioHandler) ListarPorVersion(c *gin.Context) {

	codigoProyecto, err := strconv.ParseInt(
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

	codigoCancion, err := strconv.ParseInt(
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

	codigoVersion, err := strconv.ParseInt(
		c.Param("versionId"),
		10,
		64,
	)
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

	usuario, ok := valorUsuario.(*model.Usuario)

	if !ok || usuario == nil {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "Usuario no autenticado"},
		)
		return
	}

	comentarios, err :=
		h.service.ListarPorVersion(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoCancion,
			codigoVersion,
		)

	switch {

	case errors.Is(
		err,
		service.ErrComentarioProyectoNoEncontrado,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "El proyecto no existe"},
		)

	case errors.Is(
		err,
		service.ErrComentarioCancionNoEncontrada,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "La canción no existe en el proyecto"},
		)

	case errors.Is(
		err,
		service.ErrComentarioVersionNoEncontrada,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "La versión no existe en la canción"},
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
			"Error al listar comentarios:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al obtener los comentarios"},
		)

	default:
		c.JSON(
			http.StatusOK,
			comentarios,
		)
	}
}

func (h *ComentarioHandler) Responder(c *gin.Context) {

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

	codigoComentario, err :=
		strconv.ParseInt(c.Param("comentarioId"), 10, 64)

	if err != nil || codigoComentario <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Comentario inválido"},
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

	var request dto.CrearRespuestaComentarioRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	respuesta, err :=
		h.service.Responder(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoCancion,
			codigoVersion,
			codigoComentario,
			request,
		)

	switch {

	case errors.Is(
		err,
		service.ErrRespuestaComentarioTextoObligatorio,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "La respuesta no puede estar vacía"},
		)

	case errors.Is(
		err,
		service.ErrRespuestaComentarioTextoMuyLargo,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "La respuesta no puede superar los 200 caracteres"},
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
		),
		errors.Is(
			err,
			service.ErrComentarioNoEncontrado,
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
			"Error al responder comentario:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al responder el comentario"},
		)

	default:
		c.JSON(
			http.StatusCreated,
			respuesta,
		)
	}
}

func (h *ComentarioHandler) Modificar(c *gin.Context) {

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

	codigoComentario, err :=
		strconv.ParseInt(c.Param("comentarioId"), 10, 64)

	if err != nil || codigoComentario <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Comentario inválido"},
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

	var request dto.ModificarComentarioRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Datos del comentario inválidos"},
		)
		return
	}

	comentario, err :=
		h.service.Modificar(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoCancion,
			codigoVersion,
			codigoComentario,
			request,
		)

	switch {

	case errors.Is(err, service.ErrComentarioTextoObligatorio):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El comentario no puede estar vacío"},
		)

	case errors.Is(err, service.ErrComentarioTextoMuyLargo):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El comentario no puede superar los 200 caracteres"},
		)

	case errors.Is(err, service.ErrComentarioRangoInvalido):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El rango de tiempo seleccionado no es válido"},
		)

	case errors.Is(err, service.ErrComentarioNoEncontrado):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "El comentario solicitado no existe"},
		)

	case errors.Is(err, service.ErrComentarioNoEsPropio):
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "No tenés permisos para modificar este comentario"},
		)

	case errors.Is(err, service.ErrComentarioSinAcceso):
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "No tenés acceso a este proyecto"},
		)

	case err != nil:
		log.Println(
			"Error al modificar comentario:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al modificar el comentario"},
		)

	default:
		c.JSON(
			http.StatusOK,
			comentario,
		)
	}
}

func (h *ComentarioHandler) Eliminar(c *gin.Context) {

	codigoProyecto, err :=
		strconv.ParseInt(c.Param("proyectoId"), 10, 64)

	if err != nil || codigoProyecto <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Proyecto inválido"},
		)
		return
	}

	codigoCancion, err :=
		strconv.ParseInt(c.Param("cancionId"), 10, 64)

	if err != nil || codigoCancion <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Canción inválida"},
		)
		return
	}

	codigoVersion, err :=
		strconv.ParseInt(c.Param("versionId"), 10, 64)

	if err != nil || codigoVersion <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Versión inválida"},
		)
		return
	}

	codigoComentario, err :=
		strconv.ParseInt(c.Param("comentarioId"), 10, 64)

	if err != nil || codigoComentario <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Comentario inválido"},
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

	err = h.service.Eliminar(
		usuario.CodigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
		codigoComentario,
	)

	switch {

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
		),
		errors.Is(
			err,
			service.ErrComentarioNoEncontrado,
		):

		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "El contenido solicitado no existe"},
		)

	case errors.Is(
		err,
		service.ErrComentarioSinPermisoEliminar,
	):

		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "No tenés permisos para eliminar este comentario",
			},
		)

	case errors.Is(
		err,
		service.ErrComentarioSinAcceso,
	):

		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "No tenés acceso a este proyecto",
			},
		)

	case err != nil:

		log.Println(
			"Error al eliminar comentario:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al eliminar el comentario",
			},
		)

	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *ComentarioHandler) CambiarEstado(c *gin.Context) {

	codigoProyecto, err :=
		strconv.ParseInt(c.Param("proyectoId"), 10, 64)

	if err != nil || codigoProyecto <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Proyecto inválido"},
		)
		return
	}

	codigoCancion, err :=
		strconv.ParseInt(c.Param("cancionId"), 10, 64)

	if err != nil || codigoCancion <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Canción inválida"},
		)
		return
	}

	codigoVersion, err :=
		strconv.ParseInt(c.Param("versionId"), 10, 64)

	if err != nil || codigoVersion <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Versión inválida"},
		)
		return
	}

	codigoComentario, err :=
		strconv.ParseInt(c.Param("comentarioId"), 10, 64)

	if err != nil || codigoComentario <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Comentario inválido"},
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

	var request dto.CambiarEstadoComentarioRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Estado del comentario inválido"},
		)
		return
	}

	comentario, err :=
		h.service.CambiarEstado(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoCancion,
			codigoVersion,
			codigoComentario,
			request,
		)

	switch {

	case errors.Is(
		err,
		service.ErrComentarioEstadoInvalido,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "El estado debe ser Pendiente o Resuelto",
			},
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
		),
		errors.Is(
			err,
			service.ErrComentarioNoEncontrado,
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
			"Error al cambiar estado del comentario:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al cambiar el estado del comentario",
			},
		)

	default:
		c.JSON(
			http.StatusOK,
			comentario,
		)
	}
}
