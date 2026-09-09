package router

import (
	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
)

func NewRouter(rolHandler *handler.RolHandler,
	generoMusicalHandler *handler.GeneroMusicalHandler,
	tipoProyectoHandler *handler.TipoProyectoHandler,
	usuarioHandler *handler.UsuarioHandler,
	usuarioAdministracionHandler *handler.UsuarioAdministracionHandler,
	integranteHandler *handler.IntegranteHandler,
	proyectoHandler *handler.ProyectoHandler,
	cancionHandler *handler.CancionHandler,
	comentarioHandler *handler.ComentarioHandler,
	permisoHandler *handler.PermisoHandler,
	rolPermisoHandler *handler.RolPermisoHandler,
	authMiddleware *middleware.AuthMiddleware,
	permisoMiddleware *middleware.PermisoMiddleware,
	corsAllowedOrigins []string,
) *gin.Engine {
	router := gin.Default()

	// Debe ser mayor al tamaño máximo de archivo de audio aceptado
	// (service.TamanoMaximoArchivoAudio) para no cortar el multipart antes
	// de que el handler pueda devolver un 413 controlado.
	router.MaxMultipartMemory = 110 << 20 // 110 MiB

	router.Use(middleware.Cors(corsAllowedOrigins))

	router.GET("/roles", rolHandler.Listar)

	usuarios := router.Group("/usuarios")
	usuarios.Use(authMiddleware.ValidarJWT)
	usuarios.POST("/registrar", usuarioHandler.Registrar)

	configuracion := router.Group("/configuracion")
	configuracion.Use(
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
	)

	// Catálogo de solo lectura: cualquier usuario autenticado y activo lo
	// necesita para crear un proyecto, no solo quienes tengan un rol de
	// SISTEMA. CONSULTAR_GENEROS queda reservado para gestión (crear/editar).
	configuracion.GET(
		"/generos",
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

	// Mismo criterio que /generos: catálogo de solo lectura, sin permiso.
	configuracion.GET(
		"/tipos-proyecto",
		tipoProyectoHandler.Listar,
	)

	usuarios.POST(
		"/:codigoUsuario/roles",
		authMiddleware.UsuarioActivo,
		permisoMiddleware.RequerirPermiso(
			"GESTIONAR_USUARIOS",
		),
		usuarioHandler.AsignarRol,
	)

	perfil := router.Group("/perfil")
	perfil.Use(
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
	)

	perfil.GET(
		"",
		integranteHandler.ObtenerPerfil,
	)

	perfil.PUT(
		"",
		integranteHandler.EditarPerfil,
	)

	proyectos := router.Group("/proyectos")
	proyectos.Use(
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
	)

	proyectos.POST(
		"/crear",
		proyectoHandler.Crear,
	)

	proyectos.POST(
		"/:proyectoId/canciones",
		cancionHandler.Crear,
	)

	proyectos.POST(
		"/:proyectoId/canciones/:cancionId/versiones",
		cancionHandler.CrearVersion,
	)

	proyectos.POST(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/comentarios",
		comentarioHandler.Crear,
	)

	proyectos.GET(
		"",
		proyectoHandler.Listar,
	)

	proyectos.GET(
		"/:proyectoId/integrantes",
		proyectoHandler.ListarColaboradores,
	)

	proyectos.GET(
		"/:proyectoId/canciones",
		cancionHandler.ListarPorProyecto,
	)

	proyectos.GET(
		"/:proyectoId/canciones/:cancionId/versiones",
		cancionHandler.ListarVersiones,
	)

	proyectos.GET(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/audio",
		cancionHandler.ObtenerAudioVersion,
	)

	proyectos.GET(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/comentarios",
		comentarioHandler.ListarPorVersion,
	)

	permisos := router.Group("/permisos")

	permisos.Use(
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
		permisoMiddleware.RequerirPermiso(
			"GESTIONAR_ROLES",
		),
	)

	permisos.GET(
		"",
		permisoHandler.Listar,
	)

	router.GET(
		"/roles/:id/permisos",
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
		permisoMiddleware.RequerirPermiso("GESTIONAR_ROLES"),
		rolPermisoHandler.ObtenerPorRol,
	)

	router.GET(
		"/usuarios",
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
		permisoMiddleware.RequerirPermiso("GESTIONAR_USUARIOS"),
		usuarioAdministracionHandler.Listar,
	)

	router.GET(
		"/usuarios/:id",
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
		permisoMiddleware.RequerirPermiso("GESTIONAR_USUARIOS"),
		usuarioAdministracionHandler.ObtenerPorID,
	)

	router.GET(
		"/usuarios/:id/roles",
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
		permisoMiddleware.RequerirPermiso("GESTIONAR_USUARIOS"),
		usuarioAdministracionHandler.ObtenerRolesSistema,
	)

	router.GET(
		"/usuarios/:id/proyectos",
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
		permisoMiddleware.RequerirPermiso("GESTIONAR_USUARIOS"),
		usuarioAdministracionHandler.ObtenerProyectosPorUsuario,
	)

	return router
}
