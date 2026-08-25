package main

import (
	"log"
	"os"
	"strings"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/database"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/router"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

func main() {
	db, err := database.NewPostgresConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Conexión con PostgreSQL establecida correctamente")

	//Rol
	rolRepository := repository.NewRolRepository(db)
	rolService := service.NewRolService(rolRepository)
	rolHandler := handler.NewRolHandler(rolService)

	// Género musical
	generoMusicalRepository := repository.NewGeneroMusicalRepository(db)
	generoMusicalService := service.NewGeneroMusicalService(generoMusicalRepository)
	generoMusicalHandler := handler.NewGeneroMusicalHandler(generoMusicalService)

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

	//Integrante
	integranteRepository := repository.NewIntegranteRepository(db)
	integranteService := service.NewIntegranteService(integranteRepository)
	integranteHandler := handler.NewIntegranteHandler(integranteService)

	//Proyecto
	proyectoRepository := repository.NewProyectoRepository(db)
	proyectoService := service.NewProyectoService(
		proyectoRepository,
		integranteRepository,
	)
	proyectoHandler := handler.NewProyectoHandler(proyectoService)

	//Canción
	cancionRepository := repository.NewCancionRepository(db)
	cancionService := service.NewCancionService(
		cancionRepository,
		proyectoRepository,
		integranteRepository,
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
		rolHandler,
		generoMusicalHandler,
		usuarioHandler,
		integranteHandler,
		proyectoHandler,
		cancionHandler,
		comentarioHandler,
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
