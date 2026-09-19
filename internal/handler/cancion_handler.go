package handler

import (
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

func extraerArchivoAudio(c *gin.Context) (service.ArchivoAudio, multipart.File, error) {
	fileHeader, err := c.FormFile("archivo")

	if err != nil {
		return service.ArchivoAudio{}, nil, nil
	}

	archivo, err := fileHeader.Open()

	if err != nil {
		return service.ArchivoAudio{}, nil, err
	}

	return service.ArchivoAudio{
		Contenido:      archivo,
		NombreOriginal: fileHeader.Filename,
		Tamano:         fileHeader.Size,
	}, archivo, nil
}

// extraerArchivosStems lee los stems opcionales del multipart: un campo
// repetido "stemsArchivo" (uno por archivo) emparejado por orden con
// "stemsNombre" (uno por nombre, mismo índice). Ambos deben venir en la
// misma cantidad y orden — el frontend los agrega siempre en pares.
func extraerArchivosStems(c *gin.Context) ([]service.ArchivoStem, []multipart.File, error) {

	form, err := c.MultipartForm()

	if err != nil {
		return nil, nil, nil
	}

	fileHeaders := form.File["stemsArchivo"]
	nombres := form.Value["stemsNombre"]

	if len(fileHeaders) == 0 {
		return nil, nil, nil
	}

	if len(fileHeaders) != len(nombres) {
		return nil, nil, errors.New("cada stem debe tener un nombre y un archivo")
	}

	stems := make([]service.ArchivoStem, 0, len(fileHeaders))
	archivosAbiertos := make([]multipart.File, 0, len(fileHeaders))

	for i, fileHeader := range fileHeaders {

		archivo, err := fileHeader.Open()

		if err != nil {
			return nil, archivosAbiertos, err
		}

		archivosAbiertos = append(archivosAbiertos, archivo)

		stems = append(stems, service.ArchivoStem{
			Nombre:         nombres[i],
			Contenido:      archivo,
			NombreOriginal: fileHeader.Filename,
			Tamano:         fileHeader.Size,
		})
	}

	return stems, archivosAbiertos, nil
}

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

	nombre := c.PostForm("nombre")

	archivo, archivoAbierto, err := extraerArchivoAudio(c)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	if archivoAbierto != nil {
		defer archivoAbierto.Close()
	}

	cancion, err :=
		h.service.Crear(
			usuario.CodigoUsuario,
			codigoProyecto,
			nombre,
			archivo,
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
		service.ErrCancionFormatoInvalido,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "El formato del archivo debe ser MP3, WAV o FLAC",
			},
		)

	case errors.Is(
		err,
		service.ErrCancionArchivoDemasiadoGrande,
	):
		c.JSON(
			http.StatusRequestEntityTooLarge,
			gin.H{
				"error": "El archivo de audio supera el tamaño máximo permitido",
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

	case errors.Is(
		err,
		service.ErrCancionErrorAlmacenamiento,
	):
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al almacenar el archivo de audio"},
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

	archivo, archivoAbierto, err := extraerArchivoAudio(c)

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	if archivoAbierto != nil {
		defer archivoAbierto.Close()
	}

	stems, stemsAbiertos, err := extraerArchivosStems(c)

	for _, stemAbierto := range stemsAbiertos {
		defer stemAbierto.Close()
	}

	if err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	var notas *string

	if valor, existe := c.GetPostForm("notas"); existe {
		notas = &valor
	}

	version, err :=
		h.service.CrearVersion(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoCancion,
			archivo,
			notas,
			stems,
		)
	switch {

	case errors.Is(
		err,
		service.ErrStemNombreObligatorio,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Cada stem debe tener un nombre"},
		)

	case errors.Is(
		err,
		service.ErrStemFormatoInvalido,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El formato de un stem debe ser MP3, WAV o FLAC"},
		)

	case errors.Is(
		err,
		service.ErrStemArchivoDemasiadoGrande,
	):
		c.JSON(
			http.StatusRequestEntityTooLarge,
			gin.H{"error": "Un stem supera el tamaño máximo permitido"},
		)

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
		service.ErrCancionFormatoInvalido,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "El formato del archivo debe ser MP3, WAV o FLAC",
			},
		)

	case errors.Is(
		err,
		service.ErrCancionArchivoDemasiadoGrande,
	):
		c.JSON(
			http.StatusRequestEntityTooLarge,
			gin.H{
				"error": "El archivo de audio supera el tamaño máximo permitido",
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

	case errors.Is(
		err,
		service.ErrCancionErrorAlmacenamiento,
	):
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al almacenar el archivo de audio"},
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

func (h *CancionHandler) ListarPorProyecto(
	c *gin.Context,
) {

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

	canciones, err :=
		h.service.ListarPorProyecto(
			usuario.CodigoUsuario,
			codigoProyecto,
		)

	switch {

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
		service.ErrCancionSinAccesoProyecto,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "No tenés acceso a este proyecto"},
		)

	case errors.Is(
		err,
		service.ErrCancionPerfilRequerido,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "Perfil de StemHub requerido"},
		)

	case err != nil:

		log.Println(
			"Error al listar canciones:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al obtener las canciones"},
		)

	default:
		c.JSON(
			http.StatusOK,
			canciones,
		)
	}
}

func (h *CancionHandler) ListarVersiones(
	c *gin.Context,
) {

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

	versiones, err :=
		h.service.ListarVersiones(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoCancion,
		)

	switch {

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
		service.ErrVersionCancionNoEncontrada,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "La canción no existe en este proyecto"},
		)

	case errors.Is(
		err,
		service.ErrCancionSinAccesoProyecto,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "No tenés acceso a este proyecto"},
		)

	case err != nil:
		log.Println(
			"Error al listar versiones:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al obtener las versiones"},
		)

	default:
		c.JSON(
			http.StatusOK,
			versiones,
		)
	}
}

func (h *CancionHandler) ObtenerAudioVersion(
	c *gin.Context,
) {

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

	audio, err :=
		h.service.ObtenerURLDescargaVersion(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoCancion,
			codigoVersion,
		)

	switch {

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
		service.ErrVersionCancionNoEncontrada,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "La versión no existe en esta canción"},
		)

	case errors.Is(
		err,
		service.ErrVersionSinArchivo,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "La versión no tiene un archivo de audio cargado"},
		)

	case errors.Is(
		err,
		service.ErrCancionSinAccesoProyecto,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "No tenés acceso a este proyecto"},
		)

	case errors.Is(
		err,
		service.ErrCancionErrorAlmacenamiento,
	):
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al generar la URL del archivo de audio"},
		)

	case err != nil:
		log.Println(
			"Error al obtener audio de versión:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al obtener el audio de la versión"},
		)

	default:
		c.JSON(
			http.StatusOK,
			audio,
		)
	}
}

func (h *CancionHandler) ListarMisCanciones(
	c *gin.Context,
) {

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

	canciones, err :=
		h.service.ListarMisCanciones(
			usuario.CodigoUsuario,
		)

	switch {

	case errors.Is(
		err,
		service.ErrCancionPerfilRequerido,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{"error": "Perfil de StemHub requerido"},
		)

	case err != nil:

		log.Println(
			"Error al listar mis canciones:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al obtener las canciones"},
		)

	default:
		c.JSON(
			http.StatusOK,
			canciones,
		)
	}
}
