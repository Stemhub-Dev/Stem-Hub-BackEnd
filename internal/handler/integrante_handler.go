package handler

import (
	"errors"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type IntegranteHandler struct {
	service           service.IntegranteService
	usuarioRolService service.UsuarioRolService
}

func NewIntegranteHandler(
	service service.IntegranteService,
	usuarioRolService service.UsuarioRolService,
) *IntegranteHandler {
	return &IntegranteHandler{
		service:           service,
		usuarioRolService: usuarioRolService,
	}
}

func extraerArchivoAvatar(c *gin.Context) (*service.ArchivoImagen, multipart.File, error) {
	fileHeader, err := c.FormFile("avatar")

	if err != nil {
		return nil, nil, nil
	}

	archivo, err := fileHeader.Open()

	if err != nil {
		return nil, nil, err
	}

	return &service.ArchivoImagen{
		Contenido:      archivo,
		NombreOriginal: fileHeader.Filename,
		Tamano:         fileHeader.Size,
	}, archivo, nil
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

	integrante, avatarUrl, err :=
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
		esAdministradorSistema, err :=
			h.usuarioRolService.EsAdministradorSistema(
				usuario.CodigoUsuario,
			)

		if err != nil {
			log.Println(
				"Error al consultar rol de sistema:",
				err,
			)

			c.JSON(
				http.StatusInternalServerError,
				gin.H{"error": "Error al obtener el perfil"},
			)
			return
		}

		c.JSON(
			http.StatusOK,
			dto.ObtenerPerfilResponse{
				CodigoIntegrante:       integrante.CodIntegrante,
				Email:                  usuario.Email,
				Nombre:                 integrante.NombreIntegrante,
				Descripcion:            integrante.DescripcionIntegrante,
				AvatarUrl:              avatarUrl,
				EsAdministradorSistema: esAdministradorSistema,
			},
		)
	}
}

func (h *IntegranteHandler) EditarPerfil(c *gin.Context) {

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

	var descripcion *string

	if valor, existe := c.GetPostForm("descripcion"); existe {
		descripcion = &valor
	}

	avatar, archivoAbierto, err := extraerArchivoAvatar(c)

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

	integrante, avatarUrl, err :=
		h.service.EditarPerfil(
			usuario.CodigoUsuario,
			nombre,
			descripcion,
			avatar,
		)

	switch {

	case errors.Is(
		err,
		service.ErrPerfilNombreObligatorio,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El nombre no puede estar vacío"},
		)

	case errors.Is(
		err,
		service.ErrPerfilAvatarFormatoInvalido,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "El formato de la imagen no está soportado"},
		)

	case errors.Is(
		err,
		service.ErrPerfilAvatarDemasiadoGrande,
	):
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "La imagen supera el tamaño máximo permitido"},
		)

	case errors.Is(
		err,
		service.ErrPerfilNoEncontrado,
	):
		c.JSON(
			http.StatusNotFound,
			gin.H{"error": "Perfil no encontrado"},
		)

	case errors.Is(
		err,
		service.ErrPerfilErrorAlmacenamiento,
	):
		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al almacenar el avatar"},
		)

	case err != nil:
		log.Println(
			"Error al editar perfil:",
			err,
		)

		c.JSON(
			http.StatusInternalServerError,
			gin.H{"error": "Error al editar el perfil"},
		)

	default:
		c.JSON(
			http.StatusOK,
			dto.EditarPerfilResponse{
				CodigoIntegrante: integrante.CodIntegrante,
				Email:            usuario.Email,
				Nombre:           integrante.NombreIntegrante,
				Descripcion:      integrante.DescripcionIntegrante,
				AvatarUrl:        avatarUrl,
			},
		)
	}
}
