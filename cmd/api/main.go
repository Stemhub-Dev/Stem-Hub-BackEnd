package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/database"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/mailer"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/router"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("No se encontró archivo .env, se usarán variables de entorno del sistema")
	}

	db, err := database.NewPostgresConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Conexión con PostgreSQL establecida correctamente")

	audioStorage, err := storage.NewMinioAudioStorage()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Conexión con MinIO establecida correctamente")

	//Rol
	rolRepository := repository.NewRolRepository(db)
	rolService := service.NewRolService(rolRepository)
	rolHandler := handler.NewRolHandler(rolService)

	// Género musical
	generoMusicalRepository := repository.NewGeneroMusicalRepository(db)
	generoMusicalService := service.NewGeneroMusicalService(generoMusicalRepository)
	generoMusicalHandler := handler.NewGeneroMusicalHandler(generoMusicalService)

	// Tipo de proyecto
	tipoProyectoRepository := repository.NewTipoProyectoRepository(db)
	tipoProyectoService := service.NewTipoProyectoService(tipoProyectoRepository)
	tipoProyectoHandler := handler.NewTipoProyectoHandler(tipoProyectoService)

	// Usuario y UsuarioRol
	usuarioRolRepository := repository.NewUsuarioRolRepository(db)
	usuarioRolService := service.NewUsuarioRolService(usuarioRolRepository)
	usuarioRepository := repository.NewUsuarioRepository(db)
	usuarioService := service.NewUsuarioService(usuarioRepository)
	usuarioHandler := handler.NewUsuarioHandler(usuarioService, usuarioRolService)

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAudience := os.Getenv("SUPABASE_AUDIENCE")
	// Opcional: solo necesaria en desarrollo local dockerizado, cuando el
	// backend no puede alcanzar SUPABASE_URL por red (ver supabase/README.md).
	supabaseJWKSBaseURL := os.Getenv("SUPABASE_JWKS_BASE_URL")

	authMiddleware, err := middleware.NewAuthMiddleware(
		usuarioService,
		supabaseURL,
		supabaseAudience,
		supabaseJWKSBaseURL,
	)

	if err != nil {
		log.Fatalf("error al configurar autenticación: %v", err)
	}

	// Permiso
	permisoRepository := repository.NewPermisoRepository(db)
	permisoService := service.NewPermisoService(permisoRepository)
	permisoMiddleware := middleware.NewPermisoMiddleware(permisoService)
	permisoHandler := handler.NewPermisoHandler(permisoService)

	// RolPermiso
	rolPermisoRepository := repository.NewRolPermisoRepository(db)
	rolPermisoService := service.NewRolPermisoService(rolPermisoRepository)
	rolPermisoHandler := handler.NewRolPermisoHandler(rolPermisoService)

	// UsuarioAdministracion
	usuarioAdministracionRepository := repository.NewUsuarioAdministracionRepository(db)
	usuarioAdministracionService := service.NewUsuarioAdministracionService(usuarioAdministracionRepository)
	usuarioAdministracionHandler := handler.NewUsuarioAdministracionHandler(usuarioAdministracionService)

	//Integrante
	integranteRepository := repository.NewIntegranteRepository(db)
	integranteService := service.NewIntegranteService(integranteRepository, audioStorage)
	integranteHandler := handler.NewIntegranteHandler(integranteService, usuarioRolService)

	//Proyecto
	proyectoRepository := repository.NewProyectoRepository(db)
	proyectoService := service.NewProyectoService(
		proyectoRepository,
		integranteRepository,
		audioStorage,
	)
	proyectoHandler := handler.NewProyectoHandler(proyectoService)

	//Mailer (SMTP directo, separado del auth.email.smtp de GoTrue)
	smtpMailer, err := mailer.NewSMTPMailer(
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASSWORD"),
		os.Getenv("SMTP_REMITENTE"),
	)

	if err != nil {
		log.Fatalf("error al configurar el mailer: %v", err)
	}

	//Invitación a proyecto
	diasVencimientoInvitacion, err := strconv.Atoi(os.Getenv("INVITACION_DIAS_VENCIMIENTO"))
	if err != nil {
		diasVencimientoInvitacion = 7
	}

	invitacionProyectoRepository := repository.NewInvitacionProyectoRepository(db)
	invitacionProyectoService := service.NewInvitacionProyectoService(
		invitacionProyectoRepository,
		proyectoRepository,
		integranteRepository,
		usuarioRepository,
		rolRepository,
		smtpMailer,
		os.Getenv("FRONTEND_BASE_URL"),
		diasVencimientoInvitacion,
	)
	invitacionProyectoHandler := handler.NewInvitacionProyectoHandler(invitacionProyectoService)

	//Canción
	cancionRepository := repository.NewCancionRepository(db)
	cancionService := service.NewCancionService(
		cancionRepository,
		proyectoRepository,
		integranteRepository,
		audioStorage,
	)
	cancionHandler := handler.NewCancionHandler(cancionService)

	//Comentario
	comentarioRepository := repository.NewComentarioRepository(db)
	comentarioService := service.NewComentarioService(
		comentarioRepository,
		proyectoRepository,
		cancionRepository,
		integranteRepository,
	)
	comentarioHandler := handler.NewComentarioHandler(comentarioService)

	corsAllowedOrigins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",")

	r := router.NewRouter(
		db,
		rolHandler,
		generoMusicalHandler,
		tipoProyectoHandler,
		usuarioHandler,
		usuarioAdministracionHandler,
		integranteHandler,
		proyectoHandler,
		invitacionProyectoHandler,
		cancionHandler,
		comentarioHandler,
		permisoHandler,
		rolPermisoHandler,
		authMiddleware,
		permisoMiddleware,
		corsAllowedOrigins,
	)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor iniciado en http://localhost:%s", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
