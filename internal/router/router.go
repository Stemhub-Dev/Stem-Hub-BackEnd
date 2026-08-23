package router

import (
	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
)

func NewRouter(rolHandler *handler.RolHandler,
	generoMusicalHandler *handler.GeneroMusicalHandler,
	usuarioHandler *handler.UsuarioHandler,
	authMiddleware *middleware.AuthMiddleware,
	permisoMiddleware *middleware.PermisoMiddleware,
) *gin.Engine {
	router := gin.Default()

	router.GET("/roles", rolHandler.Listar)

	usuarios := router.Group("/usuarios")
	usuarios.Use(authMiddleware.ValidarJWT)
	usuarios.POST("/registrar", usuarioHandler.Registrar)

	configuracion := router.Group("/configuracion")
	configuracion.Use(
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
	)

	configuracion.GET(
		"/generos",
		permisoMiddleware.RequerirPermiso("CONSULTAR_GENEROS"),
		generoMusicalHandler.Listar,
	)

	configuracion.POST(
		"/generos",
		permisoMiddleware.RequerirPermiso("GESTIONAR_GENEROS"),
		generoMusicalHandler.Crear,
	)

	configuracion.PUT(
		"/generos/:id",
		permisoMiddleware.RequerirPermiso("GESTIONAR_GENEROS"),
		generoMusicalHandler.Editar,
	)

	configuracion.PATCH(
		"/generos/:id",
		permisoMiddleware.RequerirPermiso("GESTIONAR_GENEROS"),
		generoMusicalHandler.CambiarEstado,
	)

	usuarios.POST(
		"/:codigoUsuario/roles",
		authMiddleware.UsuarioActivo,
		permisoMiddleware.RequerirPermiso(
			"GESTIONAR_USUARIOS",
		),
		usuarioHandler.AsignarRol,
	)

	return router
}
