package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type StemHandler struct {
	service service.StemService
}

func NewStemHandler(service service.StemService) *StemHandler {
	return &StemHandler{
		service: service,
	}
}

// rutaStem son los parámetros comunes de las rutas
// /proyectos/:proyectoId/canciones/:cancionId/versiones/:versionId/stems[/:stemId].
type rutaStem struct {
	codigoUsuario  int64
	codigoProyecto int64
	codigoCancion  int64
	codigoVersion  int64
	codStem        int64
}

// leerRutaStem responde el error y devuelve false si algún parámetro es
// inválido o no hay usuario autenticado. conStem indica si la ruta incluye
// :stemId.
func leerRutaStem(c *gin.Context, conStem bool) (rutaStem, bool) {

	var ruta rutaStem

	parametros := []struct {
		nombre  string
		destino *int64
		error   string
	}{
		{"proyectoId", &ruta.codigoProyecto, "Proyecto inválido"},
		{"cancionId", &ruta.codigoCancion, "Canción inválida"},
		{"versionId", &ruta.codigoVersion, "Versión inválida"},
	}

	if conStem {
		parametros = append(parametros, struct {
			nombre  string
			destino *int64
			error   string
		}{"stemId", &ruta.codStem, "Stem inválido"})
	}

	for _, parametro := range parametros {

		valor, err := strconv.ParseInt(c.Param(parametro.nombre), 10, 64)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": parametro.error})
			return ruta, false
		}

		*parametro.destino = valor
	}

	usuario, ok := usuarioAutenticado(c)

	if !ok {
		return ruta, false
	}

	ruta.codigoUsuario = usuario.CodigoUsuario

	return ruta, true
}

func responderErrorStem(c *gin.Context, err error) {

	type respuesta struct {
		status  int
		mensaje string
	}

	var r respuesta

	switch {
	case errors.Is(err, service.ErrStemProyectoNoEncontrado):
		r = respuesta{http.StatusNotFound, "El proyecto no existe"}
	case errors.Is(err, service.ErrStemVersionNoEncontrada):
		r = respuesta{http.StatusNotFound, "La versión no existe en esta canción"}
	case errors.Is(err, service.ErrStemNoEncontrado):
		r = respuesta{http.StatusNotFound, "El stem no existe en esta versión"}
	case errors.Is(err, service.ErrCancionPerfilRequerido):
		r = respuesta{http.StatusConflict, "Completá tu perfil en StemHub"}
	case errors.Is(err, service.ErrStemSinAcceso):
		r = respuesta{http.StatusForbidden, "No tenés acceso a este proyecto"}
	case errors.Is(err, service.ErrStemSinPermiso):
		r = respuesta{http.StatusForbidden, "No tenés permiso para gestionar stems en este proyecto"}
	case errors.Is(err, service.ErrStemNombreObligatorio):
		r = respuesta{http.StatusBadRequest, "Ingresá un nombre para el stem."}
	case errors.Is(err, service.ErrStemNombreLargo):
		r = respuesta{http.StatusBadRequest, "El nombre del stem es demasiado largo."}
	case errors.Is(err, service.ErrStemNombreDuplicado):
		r = respuesta{http.StatusConflict, "Ya existe un stem con ese nombre en esta versión."}
	case errors.Is(err, service.ErrStemCategoriaObligatoria):
		r = respuesta{http.StatusBadRequest, "Elegí una categoría."}
	case errors.Is(err, service.ErrStemCategoriaNombreLargo):
		r = respuesta{http.StatusBadRequest, "El nombre de la categoría es demasiado largo."}
	case errors.Is(err, service.ErrStemCategoriaNoEncontrada):
		r = respuesta{http.StatusBadRequest, "La categoría elegida no existe."}
	case errors.Is(err, service.ErrStemCategoriaDuplicada):
		r = respuesta{http.StatusConflict, "Esa categoría ya existe. Seleccionala de la lista."}
	case errors.Is(err, service.ErrStemArchivoObligatorio):
		r = respuesta{http.StatusBadRequest, "Subí el archivo de audio del stem."}
	case errors.Is(err, service.ErrStemArchivoVacio):
		r = respuesta{http.StatusBadRequest, "El archivo está vacío o dañado. Probá exportarlo de nuevo."}
	case errors.Is(err, service.ErrStemFormatoInvalido):
		r = respuesta{http.StatusBadRequest, "El formato del stem debe ser MP3, WAV o FLAC."}
	case errors.Is(err, service.ErrStemArchivoDemasiadoGrande):
		r = respuesta{
			http.StatusRequestEntityTooLarge,
			fmt.Sprintf("El archivo supera el máximo de %d MB.", service.TamanoMaximoArchivoStem/(1024*1024)),
		}
	case errors.Is(err, service.ErrCancionErrorAlmacenamiento):
		r = respuesta{http.StatusInternalServerError, "No se pudo guardar el archivo del stem"}
	default:
		log.Println("Error en stems:", err)
		r = respuesta{http.StatusInternalServerError, "Error al procesar el stem"}
	}

	c.JSON(r.status, gin.H{"error": r.mensaje})
}

// GET /proyectos/:proyectoId/categorias-stem
func (h *StemHandler) ListarCategorias(c *gin.Context) {

	codigoProyecto, err := strconv.ParseInt(c.Param("proyectoId"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Proyecto inválido"})
		return
	}

	usuario, ok := usuarioAutenticado(c)

	if !ok {
		return
	}

	categorias, err := h.service.ListarCategorias(usuario.CodigoUsuario, codigoProyecto)

	if err != nil {
		responderErrorStem(c, err)
		return
	}

	c.JSON(http.StatusOK, categorias)
}

// GET .../versiones/:versionId/stems
func (h *StemHandler) Listar(c *gin.Context) {

	ruta, ok := leerRutaStem(c, false)

	if !ok {
		return
	}

	stems, err := h.service.Listar(
		ruta.codigoUsuario,
		ruta.codigoProyecto,
		ruta.codigoCancion,
		ruta.codigoVersion,
	)

	if err != nil {
		responderErrorStem(c, err)
		return
	}

	c.JSON(http.StatusOK, stems)
}

// POST .../versiones/:versionId/stems — multipart/form-data con "nombre",
// "archivo" y la categoría: "codCategoriaStem" (existente) o
// "nuevaCategoria" (se crea en el proyecto). Si vienen ambas, gana
// nuevaCategoria.
func (h *StemHandler) Crear(c *gin.Context) {

	ruta, ok := leerRutaStem(c, false)

	if !ok {
		return
	}

	archivo, archivoAbierto, err := extraerArchivoAudio(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solicitud inválida"})
		return
	}

	if archivoAbierto != nil {
		defer archivoAbierto.Close()
	}

	request := service.CrearStemRequest{
		Nombre: c.PostForm("nombre"),
	}

	if valor, existe := c.GetPostForm("nuevaCategoria"); existe {
		request.NuevaCategoria = &valor
	} else if valor, existe := c.GetPostForm("codCategoriaStem"); existe {
		codigo, err := strconv.ParseInt(valor, 10, 64)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Categoría inválida"})
			return
		}

		request.CodCategoriaStem = &codigo
	}

	stem, err := h.service.Crear(
		ruta.codigoUsuario,
		ruta.codigoProyecto,
		ruta.codigoCancion,
		ruta.codigoVersion,
		request,
		archivo,
	)

	if err != nil {
		responderErrorStem(c, err)
		return
	}

	c.JSON(http.StatusCreated, stem)
}

// PUT .../stems/:stemId — multipart/form-data con "nombre" y, si se
// reemplaza el audio, "archivo". La categoría no se puede cambiar
// (HU-ABM-04-02 CA2).
func (h *StemHandler) Editar(c *gin.Context) {

	ruta, ok := leerRutaStem(c, true)

	if !ok {
		return
	}

	archivo, archivoAbierto, err := extraerArchivoAudio(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solicitud inválida"})
		return
	}

	var reemplazo *service.ArchivoAudio

	if archivoAbierto != nil {
		defer archivoAbierto.Close()
		reemplazo = &archivo
	}

	stem, err := h.service.Editar(
		ruta.codigoUsuario,
		ruta.codigoProyecto,
		ruta.codigoCancion,
		ruta.codigoVersion,
		ruta.codStem,
		c.PostForm("nombre"),
		reemplazo,
	)

	if err != nil {
		responderErrorStem(c, err)
		return
	}

	c.JSON(http.StatusOK, stem)
}

// DELETE .../stems/:stemId — borrado real de la fila y del archivo.
func (h *StemHandler) Eliminar(c *gin.Context) {

	ruta, ok := leerRutaStem(c, true)

	if !ok {
		return
	}

	if err := h.service.Eliminar(
		ruta.codigoUsuario,
		ruta.codigoProyecto,
		ruta.codigoCancion,
		ruta.codigoVersion,
		ruta.codStem,
	); err != nil {
		responderErrorStem(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GET .../stems/:stemId/audio — URL presignada para reproducir el stem.
func (h *StemHandler) ObtenerAudio(c *gin.Context) {

	ruta, ok := leerRutaStem(c, true)

	if !ok {
		return
	}

	audio, err := h.service.ObtenerURLAudio(
		ruta.codigoUsuario,
		ruta.codigoProyecto,
		ruta.codigoCancion,
		ruta.codigoVersion,
		ruta.codStem,
	)

	if err != nil {
		responderErrorStem(c, err)
		return
	}

	c.JSON(http.StatusOK, audio)
}
