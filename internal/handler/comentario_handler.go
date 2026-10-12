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
		),
		errors.Is(
			err,
			service.ErrComentarioStemNoEncontrado,
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

	ruta, ok := leerRutaComentario(c)
	if !ok {
		return
	}

	err := h.service.Eliminar(
		ruta.usuario.CodigoUsuario,
		ruta.proyecto,
		ruta.cancion,
		ruta.version,
		ruta.comentario,
	)

	responderEliminacion(c, err, mensajesEliminacion{
		noEncontrado: service.ErrComentarioNoEncontrado,
		sinPermiso:   "No tenés permisos para eliminar este comentario",
		log:          "Error al eliminar comentario:",
		errorInterno: "Error al eliminar el comentario",
	})
}

func (h *ComentarioHandler) EliminarRespuesta(c *gin.Context) {

	ruta, ok := leerRutaComentario(c)
	if !ok {
		return
	}

	codigoRespuesta, ok := parametroPositivo(c, "respuestaId", "Respuesta inválida")
	if !ok {
		return
	}

	err := h.service.EliminarRespuesta(
		ruta.usuario.CodigoUsuario,
		ruta.proyecto,
		ruta.cancion,
		ruta.version,
		ruta.comentario,
		codigoRespuesta,
	)

	responderEliminacion(c, err, mensajesEliminacion{
		noEncontrado: service.ErrRespuestaComentarioNoEncontrada,
		sinPermiso:   "No tenés permisos para eliminar esta respuesta",
		log:          "Error al eliminar respuesta de comentario:",
		errorInterno: "Error al eliminar la respuesta",
	})
}

// Ids de .../proyectos/:proyectoId/canciones/:cancionId/versiones/:versionId/
// comentarios/:comentarioId y el usuario autenticado.
type rutaComentario struct {
	usuario    *model.Usuario
	proyecto   int64
	cancion    int64
	version    int64
	comentario int64
}

// Lee la ruta de un comentario. Si algo falta o es inválido ya respondió
// (400 o 401) y devuelve false.
func leerRutaComentario(c *gin.Context) (rutaComentario, bool) {

	var ruta rutaComentario

	for _, parametro := range []struct {
		nombre  string
		mensaje string
		destino *int64
	}{
		{"proyectoId", "Proyecto inválido", &ruta.proyecto},
		{"cancionId", "Canción inválida", &ruta.cancion},
		{"versionId", "Versión inválida", &ruta.version},
		{"comentarioId", "Comentario inválido", &ruta.comentario},
	} {
		valor, ok := parametroPositivo(c, parametro.nombre, parametro.mensaje)
		if !ok {
			return ruta, false
		}
		*parametro.destino = valor
	}

	valorUsuario, existe := c.Get(middleware.UsuarioContextKey)
	usuario, ok := valorUsuario.(*model.Usuario)

	if !existe || !ok || usuario == nil {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{"error": "Usuario no autenticado"},
		)
		return ruta, false
	}

	ruta.usuario = usuario

	return ruta, true
}

func parametroPositivo(c *gin.Context, nombre string, mensaje string) (int64, bool) {

	valor, err := strconv.ParseInt(c.Param(nombre), 10, 64)

	if err != nil || valor <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": mensaje},
		)
		return 0, false
	}

	return valor, true
}

type mensajesEliminacion struct {
	// Error del service que significa "no existe" (404) para este recurso.
	noEncontrado error
	sinPermiso   string
	log          string
	errorInterno string
}

// Traduce el resultado de eliminar un comentario o una respuesta a la
// respuesta HTTP (204, 404, 403 o 500).
func responderEliminacion(c *gin.Context, err error, mensajes mensajesEliminacion) {

	switch {

	case errors.Is(err, service.ErrComentarioProyectoNoEncontrado),
		errors.Is(err, service.ErrComentarioCancionNoEncontrada),
		errors.Is(err, service.ErrComentarioVersionNoEncontrada),
		errors.Is(err, mensajes.noEncontrado):

		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "El contenido solicitado no existe"},
		)

	case errors.Is(err, service.ErrComentarioSinPermisoEliminar):

		c.JSON(
			http.StatusForbidden,
			gin.H{"error": mensajes.sinPermiso},
		)

	case errors.Is(err, service.ErrComentarioSinAcceso):

		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "No tenés acceso a este proyecto"},
		)

	case err != nil:

		log.Println(mensajes.log, err)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": mensajes.errorInterno},
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

// GET .../versiones/:versionId/comentarios/resumen — resumen de los
// comentarios generado con IA. Puede tardar varios segundos: se genera en
// cada pedido.
func (h *ComentarioHandler) ResumirPorVersion(c *gin.Context) {

	parametros := []struct {
		nombre string
		error  string
	}{
		{"proyectoId", "Proyecto inválido"},
		{"cancionId", "Canción inválida"},
		{"versionId", "Versión inválida"},
	}

	codigos := make([]int64, len(parametros))

	for i, parametro := range parametros {

		valor, err := strconv.ParseInt(c.Param(parametro.nombre), 10, 64)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": parametro.error})
			return
		}

		codigos[i] = valor
	}

	usuario, ok := usuarioAutenticado(c)

	if !ok {
		return
	}

	resumen, err := h.service.ResumirPorVersion(
		usuario.CodigoUsuario,
		codigos[0],
		codigos[1],
		codigos[2],
	)

	switch {
	case errors.Is(err, service.ErrComentarioProyectoNoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"error": "El proyecto no existe"})
	case errors.Is(err, service.ErrComentarioCancionNoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"error": "La canción no existe en el proyecto"})
	case errors.Is(err, service.ErrComentarioVersionNoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"error": "La versión no existe en la canción"})
	case errors.Is(err, service.ErrComentarioSinAcceso):
		c.JSON(http.StatusForbidden, gin.H{"error": "No tenés acceso a este proyecto"})
	case errors.Is(err, service.ErrResumenSinComentarios):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "La versión todavía no tiene comentarios para resumir"})
	case errors.Is(err, service.ErrResumenTimeout):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "El resumen tardó demasiado. Probá de nuevo en unos minutos."})
	case errors.Is(err, service.ErrResumenNoDisponible):
		c.JSON(http.StatusBadGateway, gin.H{"error": "No se pudo generar el resumen. Probá de nuevo en unos minutos."})
	case err != nil:
		log.Println("Error al resumir comentarios:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al generar el resumen"})
	default:
		c.JSON(http.StatusOK, resumen)
	}
}
