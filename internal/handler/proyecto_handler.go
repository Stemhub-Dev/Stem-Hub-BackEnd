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

// tamanoMaximoFormularioProyecto acota el cuerpo del alta y la modificación
// de un proyecto: la portada más 1 MiB para los demás campos y el overhead
// del multipart.
const tamanoMaximoFormularioProyecto = service.TamanoMaximoLogoProyecto + 1<<20

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

	var logo *service.ArchivoImagen

	// Sin portada alcanza con JSON; con portada el formulario llega como
	// multipart, con los mismos campos más el archivo "logo".
	if strings.HasPrefix(
		c.GetHeader("Content-Type"),
		"multipart/form-data",
	) {

		if !parsearFormularioProyecto(c) {
			return
		}

		var ok bool

		request, ok = leerCrearProyectoForm(c)

		if !ok {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Solicitud inválida"},
			)
			return
		}

		var archivoAbierto multipart.File
		var err error

		logo, archivoAbierto, err = extraerArchivoLogo(c)

		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				gin.H{"error": "Logo inválido"},
			)
			return
		}

		if archivoAbierto != nil {
			defer archivoAbierto.Close()
		}

	} else if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	proyecto, err := h.service.CrearProyecto(
		usuario.CodigoUsuario,
		request,
		logo,
	)

	if responderErrorLogoProyecto(c, err) {
		return
	}

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

// parsearFormularioProyecto lee el formulario con el cuerpo acotado, antes
// de que cualquier PostForm lo parsee entero: sin el límite, un archivo
// enorme se escribiría a disco antes de que el service lo rechace. Todo lo
// permitido entra en memoria. Devuelve false si ya respondió con un error.
func parsearFormularioProyecto(
	c *gin.Context,
) bool {

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		tamanoMaximoFormularioProyecto,
	)

	err := c.Request.ParseMultipartForm(
		tamanoMaximoFormularioProyecto,
	)

	if errors.Is(err, http.ErrNotMultipart) {
		err = c.Request.ParseForm()
	}

	var demasiadoGrande *http.MaxBytesError

	switch {

	case errors.As(err, &demasiadoGrande):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "La imagen supera el tamaño máximo permitido"},
		)
		return false

	case err != nil:
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return false
	}

	return true
}

// leerCrearProyectoForm arma el request de alta desde un multipart, con las
// mismas reglas que los binding del JSON: nombre, tipo, rol y al menos un
// género obligatorios.
func leerCrearProyectoForm(
	c *gin.Context,
) (dto.CrearProyectoRequest, bool) {

	var request dto.CrearProyectoRequest

	request.Nombre = c.PostForm("nombre")

	if valor, existe :=
		c.GetPostForm("descripcion"); existe {

		request.Descripcion = &valor
	}

	codigoTipo, err := strconv.ParseInt(
		c.PostForm("codigoTipoProyecto"),
		10,
		64,
	)

	if err != nil || codigoTipo <= 0 {
		return request, false
	}

	request.CodigoTipoProyecto = codigoTipo

	codRol, err := strconv.ParseInt(
		c.PostForm("codRol"),
		10,
		64,
	)

	if err != nil || codRol <= 0 {
		return request, false
	}

	request.CodRol = codRol

	for _, valor := range c.PostFormArray("codigosGeneros") {

		codigoGenero, err := strconv.ParseInt(valor, 10, 64)

		if err != nil || codigoGenero <= 0 {
			return request, false
		}

		request.CodigosGeneros = append(
			request.CodigosGeneros,
			codigoGenero,
		)
	}

	if strings.TrimSpace(request.Nombre) == "" ||
		len(request.CodigosGeneros) == 0 {
		return request, false
	}

	return request, true
}

// responderErrorLogoProyecto responde los errores de la portada, comunes al
// alta y a la modificación. Devuelve true si respondió.
func responderErrorLogoProyecto(
	c *gin.Context,
	err error,
) bool {

	switch {

	case errors.Is(
		err,
		service.ErrProyectoLogoFormatoInvalido,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El formato de la imagen no está soportado"},
		)

	case errors.Is(
		err,
		service.ErrProyectoLogoDemasiadoGrande,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "La imagen supera el tamaño máximo permitido"},
		)

	case errors.Is(
		err,
		service.ErrProyectoErrorAlmacenamiento,
	):
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al almacenar el logo del proyecto"},
		)

	default:
		return false
	}

	return true
}

// parsearFiltroProyectos arma el filtro de GET /proyectos a partir de q,
// estadoId, tipoId (repetibles, códigos de /configuracion/estados-proyecto y
// /configuracion/tipos-proyecto), page y pageSize. Devuelve un mensaje de
// error si algún valor es inválido.
func parsearFiltroProyectos(
	c *gin.Context,
) (dto.ListarProyectosFiltro, string) {

	estados, mensajeError := parsearCodigos(c, "estadoId")

	if mensajeError != "" {
		return dto.ListarProyectosFiltro{}, mensajeError
	}

	tipos, mensajeError := parsearCodigos(c, "tipoId")

	if mensajeError != "" {
		return dto.ListarProyectosFiltro{}, mensajeError
	}

	pagina, tamanoPagina, mensajeError := parsearPaginacion(c)

	if mensajeError != "" {
		return dto.ListarProyectosFiltro{}, mensajeError
	}

	return dto.ListarProyectosFiltro{
		Busqueda:     strings.TrimSpace(c.Query("q")),
		Estados:      estados,
		Tipos:        tipos,
		Pagina:       pagina,
		TamanoPagina: tamanoPagina,
	}, ""
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

	filtro, mensajeError :=
		parsearFiltroProyectos(c)

	if mensajeError != "" {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": mensajeError},
		)
		return
	}

	proyectos, err :=
		h.service.ListarProyectos(
			usuario.CodigoUsuario,
			filtro,
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
	var eliminarLogo bool

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

		if !parsearFormularioProyecto(c) {
			return
		}

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

		// Quitar la portada: se ignora si también viene un logo nuevo.
		eliminarLogo =
			c.PostForm("eliminarLogo") == "true"
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
			eliminarLogo,
		)

	if responderErrorLogoProyecto(c, err) {
		return
	}

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

func (h *ProyectoHandler) ListarMiParticipacion(
	c *gin.Context,
) {

	// 1. Usuario autenticado
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

	// 2. Filtro opcional
	participacion :=
		c.Query("participacion")

	// Si no viene, el service lo interpreta como "todos".

	proyectos, err :=
		h.service.ListarMiParticipacion(
			usuario.CodigoUsuario,
			participacion,
		)

	switch {

	case errors.Is(
		err,
		service.ErrParticipacionProyectoNoValida,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "La participación indicada no es válida",
			},
		)

	case errors.Is(
		err,
		service.ErrProyectoSinAcceso,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "No tenés un perfil activo de Stem-Hub",
			},
		)

	case err != nil:

		log.Println(
			"Error al listar proyectos por participación:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al obtener los proyectos",
			},
		)

	default:

		c.JSON(
			http.StatusOK,
			proyectos,
		)
	}
}

func (h *ProyectoHandler) CambiarRolIntegrante(
	c *gin.Context,
) {

	// Usuario autenticado.
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

	// Proyecto.
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

	// Integrante objetivo.
	codigoIntegrante, err :=
		strconv.ParseInt(
			c.Param("integranteId"),
			10,
			64,
		)

	if err != nil || codigoIntegrante <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Integrante inválido"},
		)
		return
	}

	var request dto.CambiarRolIntegranteProyectoRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Solicitud inválida"},
		)
		return
	}

	if request.CodRol <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El rol es obligatorio"},
		)
		return
	}

	colaborador, err :=
		h.service.CambiarRolIntegrante(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoIntegrante,
			request,
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
		service.ErrGestionIntegrantesSoloPropietario,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "Solo el propietario puede gestionar los integrantes del proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrIntegranteProyectoNoEncontrado,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": "El integrante no pertenece activamente al proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrRolProyectoNoValido,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "El rol seleccionado no es válido para un proyecto",
			},
		)

	case err != nil:
		log.Println(
			"Error al cambiar rol del integrante:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al cambiar el rol del integrante",
			},
		)

	default:
		c.JSON(
			http.StatusOK,
			colaborador,
		)
	}
}

func (h *ProyectoHandler) QuitarIntegrante(
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

	codigoIntegrante, err :=
		strconv.ParseInt(
			c.Param("integranteId"),
			10,
			64,
		)

	if err != nil || codigoIntegrante <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "Integrante inválido"},
		)
		return
	}

	err =
		h.service.QuitarIntegrante(
			usuario.CodigoUsuario,
			codigoProyecto,
			codigoIntegrante,
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
		service.ErrGestionIntegrantesSoloPropietario,
	):
		c.JSON(
			http.StatusForbidden,
			gin.H{
				"error": "Solo el propietario puede gestionar los integrantes del proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrIntegranteProyectoNoEncontrado,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{
				"error": "El integrante no pertenece activamente al proyecto",
			},
		)

	case errors.Is(
		err,
		service.ErrNoSePuedeQuitarPropietario,
	):
		c.JSON(
			http.StatusConflict,
			gin.H{
				"error": "No se puede quitar al propietario del proyecto",
			},
		)

	case err != nil:
		log.Println(
			"Error al quitar integrante del proyecto:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"error": "Error al quitar el integrante del proyecto",
			},
		)

	default:
		c.Status(http.StatusNoContent)
	}
}
