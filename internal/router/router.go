package router

import (
	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
)

func NewRouter(rolHandler *handler.RolHandler) *gin.Engine {
	router := gin.Default()

	router.GET("/roles", rolHandler.Listar)

	return router
}
