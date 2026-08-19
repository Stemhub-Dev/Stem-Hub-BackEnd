package router

import (
	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
)

func NewRouter(rolHandler *handler.RolHandler,
	generoMusicalHandler *handler.GeneroMusicalHandler,
) *gin.Engine {
	router := gin.Default()

	router.GET("/roles", rolHandler.Listar)

	configuracion := router.Group("/configuracion")
	configuracion.GET("/generos", generoMusicalHandler.Listar)
	configuracion.POST("/generos", generoMusicalHandler.Crear)
	configuracion.PUT("/generos/:id", generoMusicalHandler.Editar)
	configuracion.PATCH("/generos/:id", generoMusicalHandler.CambiarEstado)

	return router
}
