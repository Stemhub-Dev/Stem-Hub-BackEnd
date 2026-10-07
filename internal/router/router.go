package router

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
)

func NewRouter(db *sql.DB,
	rolHandler *handler.RolHandler,
	generoMusicalHandler *handler.GeneroMusicalHandler,
	tipoProyectoHandler *handler.TipoProyectoHandler,
	estadoProyectoHandler *handler.EstadoProyectoHandler,
	usuarioHandler *handler.UsuarioHandler,
	usuarioAdministracionHandler *handler.UsuarioAdministracionHandler,
	integranteHandler *handler.IntegranteHandler,
	proyectoHandler *handler.ProyectoHandler,
	invitacionProyectoHandler *handler.InvitacionProyectoHandler,
	cancionHandler *handler.CancionHandler,
	comentarioHandler *handler.ComentarioHandler,
	stemHandler *handler.StemHandler,
	reporteHandler *handler.ReporteHandler,
	tableroHandler *handler.TableroHandler,
	permisoHandler *handler.PermisoHandler,
	rolPermisoHandler *handler.RolPermisoHandler,
	authMiddleware *middleware.AuthMiddleware,
	permisoMiddleware *middleware.PermisoMiddleware,
	corsAllowedOrigins []string,
) *gin.Engine {
	router := gin.Default()

	// Cuánto del multipart se guarda en memoria; lo que excede va a un
	// archivo temporal. No es un límite de tamaño: ese lo aplican los
	// services (TAMANO_MAXIMO_CANCION_MB / TAMANO_MAXIMO_STEM_MB) con un 413.
	router.MaxMultipartMemory = 110 << 20 // 110 MiB

	router.Use(middleware.Cors(corsAllowedOrigins))

	// Sin auth a propósito: lo consulta el healthcheck de Docker Compose
	// (ver docker-compose.prod.yml), no un cliente autenticado. Verifica el
	// pool de conexiones a Postgres, no solo que el proceso Go esté vivo —
	// el puerto HTTP queda escuchando apenas arranca Gin, antes de que el
	// pool haya abierto ninguna conexión real (database/sql las abre
	// perezosamente, bajo demanda), así que sin este chequeo el proxy podría
	// enrutarle tráfico al backend en esa ventana y las primeras requests
	// fallarían con 500 mientras el pool recién se establece.
	router.GET("/health", func(c *gin.Context) {
		if err := db.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sin conexión a la base de datos"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

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

	configuracion.POST(
		"/tipos-proyecto",
		permisoMiddleware.RequerirPermiso(
			"GESTIONAR_TIPOS_PROYECTO",
		),
		tipoProyectoHandler.Crear,
	)

	configuracion.PUT(
		"/tipos-proyecto/:id",
		permisoMiddleware.RequerirPermiso(
			"GESTIONAR_TIPOS_PROYECTO",
		),
		tipoProyectoHandler.Editar,
	)

	configuracion.PATCH(
		"/tipos-proyecto/:id",
		permisoMiddleware.RequerirPermiso(
			"GESTIONAR_TIPOS_PROYECTO",
		),
		tipoProyectoHandler.CambiarEstado,
	)

	configuracion.GET(
		"/estados-proyecto",
		estadoProyectoHandler.Listar,
	)

	configuracion.POST(
		"/estados-proyecto",
		permisoMiddleware.RequerirPermiso(
			"GESTIONAR_ESTADOS_PROYECTO",
		),
		estadoProyectoHandler.Crear,
	)

	configuracion.PUT(
		"/estados-proyecto/:id",
		permisoMiddleware.RequerirPermiso(
			"GESTIONAR_ESTADOS_PROYECTO",
		),
		estadoProyectoHandler.Editar,
	)

	configuracion.PATCH(
		"/estados-proyecto/:id",
		permisoMiddleware.RequerirPermiso(
			"GESTIONAR_ESTADOS_PROYECTO",
		),
		estadoProyectoHandler.CambiarEstado,
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

	proyectos.GET(
		"/mi-participacion",
		proyectoHandler.ListarMiParticipacion,
	)

	proyectos.POST(
		"/crear",
		proyectoHandler.Crear,
	)

	proyectos.POST(
		"/:proyectoId/canciones",
		cancionHandler.Crear,
	)

	proyectos.PUT(
		"/:proyectoId/canciones/:cancionId",
		cancionHandler.Editar,
	)

	proyectos.DELETE(
		"/:proyectoId/canciones/:cancionId",
		cancionHandler.DarDeBaja,
	)

	proyectos.POST(
		"/:proyectoId/canciones/:cancionId/versiones",
		cancionHandler.CrearVersion,
	)

	proyectos.DELETE(
		"/:proyectoId",
		proyectoHandler.DarDeBaja,
	)

	proyectos.POST(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/comentarios",
		comentarioHandler.Crear,
	)

	proyectos.POST(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/comentarios/:comentarioId/respuestas",
		comentarioHandler.Responder,
	)

	proyectos.PATCH(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/comentarios/:comentarioId",
		comentarioHandler.Modificar,
	)

	proyectos.PATCH(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/comentarios/:comentarioId/estado",
		comentarioHandler.CambiarEstado,
	)

	proyectos.DELETE(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/comentarios/:comentarioId",
		comentarioHandler.Eliminar,
	)

	proyectos.GET(
		"",
		proyectoHandler.Listar,
	)

	proyectos.GET(
		"/:proyectoId",
		proyectoHandler.ObtenerDetalle,
	)

	proyectos.PATCH(
		"/:proyectoId",
		proyectoHandler.Editar,
	)

	proyectos.GET(
		"/:proyectoId/integrantes",
		proyectoHandler.ListarColaboradores,
	)

	proyectos.POST(
		"/:proyectoId/invitaciones",
		invitacionProyectoHandler.Crear,
	)

	proyectos.GET(
		"/:proyectoId/invitaciones",
		invitacionProyectoHandler.ListarPendientes,
	)

	proyectos.DELETE(
		"/:proyectoId/invitaciones/:invitacionId",
		invitacionProyectoHandler.Cancelar,
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

	// Stems (HU-ABM-04-*). Leer es para cualquier integrante del proyecto;
	// el ABM exige GESTIONAR_STEMS, que valida el service contra el rol del
	// usuario en el proyecto (no es un permiso de SISTEMA, por eso no pasa
	// por permisoMiddleware).
	proyectos.GET(
		"/:proyectoId/categorias-stem",
		stemHandler.ListarCategorias,
	)

	proyectos.GET(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/stems",
		stemHandler.Listar,
	)

	proyectos.POST(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/stems",
		stemHandler.Crear,
	)

	proyectos.PUT(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/stems/:stemId",
		stemHandler.Editar,
	)

	proyectos.DELETE(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/stems/:stemId",
		stemHandler.Eliminar,
	)

	proyectos.GET(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/stems/:stemId/audio",
		stemHandler.ObtenerAudio,
	)

	// "Separar Pistas" con IA: se procesa en segundo plano, el GET devuelve
	// el estado de la última separación. Separar exige GESTIONAR_STEMS.
	proyectos.POST(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/stems/separacion",
		stemHandler.SolicitarSeparacion,
	)

	proyectos.GET(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/stems/separacion",
		stemHandler.ObtenerSeparacion,
	)

	// Resumen de los comentarios de la versión generado con IA. Se calcula
	// en cada pedido, no se guarda.
	proyectos.GET(
		"/:proyectoId/canciones/:cancionId/versiones/:versionId/comentarios/resumen",
		comentarioHandler.ResumirPorVersion,
	)

	reportes := router.Group("/report/reportes")
	reportes.Use(
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
	)
	reportes.GET("", reporteHandler.Listar)
	reportes.POST("/:tipo/generar", reporteHandler.Generar)
	reportes.POST("/:tipo/exportar-pdf", reporteHandler.ExportarPDF)

	// Tablero (HU-DASH-B01/B02): cualquier usuario autenticado; los datos se
	// limitan a los proyectos en los que participa (lo resuelve el service).
	// Un endpoint por gráfico para que cada uno cargue y falle por separado.
	tablero := router.Group("/tablero")
	tablero.Use(
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
	)
	tablero.GET("/indicadores", tableroHandler.ObtenerIndicadores)
	tablero.GET("/graficos/versiones-por-proyecto", tableroHandler.ObtenerVersionesPorProyecto)
	tablero.GET("/graficos/canciones-por-etapa", tableroHandler.ObtenerCancionesPorEtapa)
	tablero.GET("/graficos/actividad", tableroHandler.ObtenerActividad)

	router.GET(
		"/canciones",
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
		cancionHandler.ListarMisCanciones,
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

	router.PATCH(
		"/usuarios/:id/administracion",
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
		permisoMiddleware.RequerirPermiso("GESTIONAR_USUARIOS"),
		usuarioAdministracionHandler.ActualizarAdministracion,
	)

	router.PATCH(
		"/usuarios/:id/estado",
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
		permisoMiddleware.RequerirPermiso("GESTIONAR_USUARIOS"),
		usuarioAdministracionHandler.CambiarEstado,
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

	// Público: debe poder mostrar el detalle de la invitación (proyecto,
	// quién invita, rol) antes de que la persona tenga sesión iniciada.
	router.GET(
		"/invitaciones/:token",
		invitacionProyectoHandler.ObtenerDetalle,
	)

	invitaciones := router.Group("/invitaciones")
	invitaciones.Use(
		authMiddleware.ValidarJWT,
		authMiddleware.UsuarioActivo,
	)

	invitaciones.POST(
		"/:token/aceptar",
		invitacionProyectoHandler.Aceptar,
	)

	invitaciones.POST(
		"/:token/rechazar",
		invitacionProyectoHandler.Rechazar,
	)

	// Bandeja de notificaciones: invitaciones pendientes del usuario
	// autenticado por su propio email.
	invitaciones.GET(
		"",
		invitacionProyectoHandler.MisInvitaciones,
	)

	return router
}
