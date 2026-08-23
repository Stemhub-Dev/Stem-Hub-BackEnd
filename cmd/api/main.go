package main

import (
	"log"
	"os"

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

	usuarioRepository := repository.NewUsuarioRepository(db)
	usuarioService := service.NewUsuarioService(usuarioRepository)
	usuarioHandler := handler.NewUsuarioHandler(usuarioService)

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAudience := os.Getenv("SUPABASE_AUDIENCE")

	authMiddleware, err := middleware.NewAuthMiddleware(
		usuarioService,
		supabaseURL,
		supabaseAudience,
	)

	if err != nil {
		log.Fatalf("error al configurar autenticación: %v", err)
	}

	r := router.NewRouter(
		rolHandler,
		generoMusicalHandler,
		usuarioHandler,
		authMiddleware,
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
