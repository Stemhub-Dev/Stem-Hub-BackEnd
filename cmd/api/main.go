package main

import (
	"log"
	"os"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/database"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
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

	r := router.NewRouter(
		rolHandler,
		generoMusicalHandler,
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
