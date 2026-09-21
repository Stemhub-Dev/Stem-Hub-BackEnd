package handler

import (
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

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

func extraerArchivoLogo(
	c *gin.Context,
) (
	*service.ArchivoImagen,
	multipart.File,
	error,
) {

	contentType :=
		c.GetHeader("Content-Type")

	// Si el request no es multipart,
	// significa que no viene un archivo.
	if !strings.HasPrefix(
		contentType,
		"multipart/form-data",
	) {
		return nil, nil, nil
	}

	fileHeader, err :=
		c.FormFile("logo")

	if err != nil {

		// El campo logo simplemente no fue enviado.
		if errors.Is(
			err,
			http.ErrMissingFile,
		) {
			return nil, nil, nil
		}

		return nil, nil, err
	}

	archivo, err :=
		fileHeader.Open()

	if err != nil {
		return nil, nil, err
	}

	return &service.ArchivoImagen{
		Contenido: archivo,

		NombreOriginal: fileHeader.Filename,

		Tamano: fileHeader.Size,
	}, archivo, nil
}

func (h *ProyectoHandler) ObtenerDetalle(
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

	codigoProyecto, err :=
		strconv.ParseInt(
			c.Param("proyectoId"),
			10,
			64,
		)

	if err != nil || codigoProyecto <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Proyecto inválido"},
		)
		return
	}

	proyecto, err :=
		h.service.ObtenerDetalleProyecto(
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
			"Error al obtener detalle del proyecto:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al obtener el proyecto",
			},
		)

	default:
		c.JSON(
			http.StatusOK,
			proyecto,
		)
	}
}

func (h *ProyectoHandler) Editar(c *gin.Context) {

	// =====================================================
	// 1. OBTENER USUARIO AUTENTICADO
	// =====================================================

	valorUsuario, existe :=
		c.Get(middleware.UsuarioContextKey)

	if !existe {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{
				"error": "Usuario no autenticado",
			},
		)
		return
	}

	usuario, ok :=
		valorUsuario.(*model.Usuario)

	if !ok || usuario == nil {
		c.JSON(
			http.StatusUnauthorized,
			gin.H{
				"error": "Usuario no autenticado",
			},
		)
		return
	}

	// =====================================================
	// 2. OBTENER ID DEL PROYECTO
	// =====================================================

	codigoProyecto, err :=
		strconv.ParseInt(
			c.Param("proyectoId"),
			10,
			64,
		)

	if err != nil || codigoProyecto <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Proyecto inválido",
			},
		)
		return
	}

	// =====================================================
	// 3. REQUEST DE MODIFICACIÓN
	// =====================================================

	var request dto.EditarProyectoRequest

	var logo *service.ArchivoImagen
	var archivoAbierto multipart.File

	contentType :=
		c.GetHeader("Content-Type")

	// =====================================================
	// 4. SI VIENE JSON
	// =====================================================

	if strings.HasPrefix(
		contentType,
		"application/json",
	) {

		if err :=
			c.ShouldBindJSON(&request); err != nil {

			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": "Datos de modificación inválidos",
				},
			)
			return
		}

	} else {

		// =================================================
		// 5. FORM-DATA / X-WWW-FORM-URLENCODED
		// =================================================

		// ------------------------------
		// Nombre opcional
		// ------------------------------

		if valor, existe :=
			c.GetPostForm("nombre"); existe {

			request.Nombre = &valor
		}

		// ------------------------------
		// Descripción opcional
		// ------------------------------

		if valor, existe :=
			c.GetPostForm("descripcion"); existe {

			request.Descripcion = &valor
		}

		// ------------------------------
		// Tipo de proyecto opcional
		// ------------------------------

		if valor, existe :=
			c.GetPostForm(
				"codigoTipoProyecto",
			); existe {

			codigo, err :=
				strconv.ParseInt(
					valor,
					10,
					64,
				)

			if err != nil || codigo <= 0 {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"error": "El tipo de proyecto es inválido",
					},
				)
				return
			}

			request.CodigoTipoProyecto =
				&codigo
		}

		// ------------------------------
		// Estado opcional
		// ------------------------------

		if valor, existe :=
			c.GetPostForm(
				"codigoEstadoProyecto",
			); existe {

			codigo, err :=
				strconv.ParseInt(
					valor,
					10,
					64,
				)

			if err != nil || codigo <= 0 {

				c.JSON(
					http.StatusBadRequest,
					gin.H{
						"error": "El estado del proyecto es inválido",
					},
				)
				return
			}

			request.CodigoEstadoProyecto =
				&codigo
		}

		// ------------------------------
		// Géneros opcionales
		// ------------------------------

		valoresGeneros :=
			c.PostFormArray(
				"codigosGeneros",
			)

		if len(valoresGeneros) > 0 {

			request.CodigosGeneros =
				make(
					[]int64,
					0,
					len(valoresGeneros),
				)

			for _, valorGenero := range valoresGeneros {

				codigoGenero, err :=
					strconv.ParseInt(
						valorGenero,
						10,
						64,
					)

				if err != nil ||
					codigoGenero <= 0 {

					c.JSON(
						http.StatusBadRequest,
						gin.H{
							"error": "Uno o más géneros son inválidos",
						},
					)
					return
				}

				request.CodigosGeneros =
					append(
						request.CodigosGeneros,
						codigoGenero,
					)
			}
		}

		// ------------------------------
		// Logo opcional
		// ------------------------------

		logo,
			archivoAbierto,
			err =
			extraerArchivoLogo(c)

		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{
					"error": "Logo inválido",
				},
			)
			return
		}

		if archivoAbierto != nil {
			defer archivoAbierto.Close()
		}
	}

	// =====================================================
	// 6. LLAMAR AL SERVICE
	// =====================================================

	proyecto, err :=
		h.service.EditarProyecto(
			usuario.CodigoUsuario,
			codigoProyecto,
			request,
			logo,
		)

	// =====================================================
	// 7. MANEJO DE ERRORES
	// =====================================================

	switch {

	case errors.Is(
		err,
		service.ErrNombreProyectoObligatorio,
	):

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "El nombre del proyecto es obligatorio",
			},
		)

	case errors.Is(
		err,
		service.ErrTipoProyectoNoValido,
	):

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "El tipo de proyecto no es válido",
			},
		)

	case errors.Is(
		err,
		service.ErrEstadoProyectoNoValido,
	):

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "El estado del proyecto no es válido",
			},
		)

	case errors.Is(
		err,
		service.ErrGenerosNoValidos,
	):

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Uno o más géneros no son válidos",
			},
		)

	case errors.Is(
		err,
		service.ErrProyectoLogoFormatoInvalido,
	):

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "El formato del logo no está soportado",
			},
		)

	case errors.Is(
		err,
		service.ErrProyectoLogoDemasiadoGrande,
	):

		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "El logo supera el tamaño máximo permitido",
			},
		)

	case errors.Is(
		err,
		service.ErrProyectoNoEncontrado,
	):

		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": "El proyecto no existe",
			},
		)

	case errors.Is(
		err,
		service.ErrProyectoSoloPropietario,
	):

		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "Solo el propietario puede modificar el proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrProyectoSinAcceso,
	):

		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "No tenés acceso a este proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrProyectoErrorAlmacenamiento,
	):

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al almacenar el logo del proyecto",
			},
		)

	case err != nil:

		log.Println(
			"Error al editar proyecto:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al modificar el proyecto",
			},
		)

	default:

		c.JSON(
			http.StatusOK,
			proyecto,
		)
	}
}

func (h *ProyectoHandler) DarDeBaja(
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

	codigoProyecto, err :=
		strconv.ParseInt(
			c.Param("proyectoId"),
			10,
			64,
		)

	if err != nil || codigoProyecto <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Proyecto inválido"},
		)
		return
	}

	err = h.service.DarDeBaja(
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
			gin.H{
				"error": "El proyecto no existe",
			},
		)

	case errors.Is(
		err,
		service.ErrProyectoSoloPropietario,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "Solo el propietario puede dar de baja el proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrProyectoSinAcceso,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "No tenés acceso a este proyecto",
			},
		)

	case err != nil:
		log.Println(
			"Error al dar de baja proyecto:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al dar de baja el proyecto",
			},
		)

	default:
		c.JSON(
			http.StatusOK,
			gin.H{
				"mensaje": "Proyecto dado de baja correctamente",
			},
		)
	}
}
