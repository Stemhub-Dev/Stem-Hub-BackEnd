package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

type PreguntaFrecuenteHandler struct {
	service *service.PreguntaFrecuenteService
}

func NewPreguntaFrecuenteHandler(service *service.PreguntaFrecuenteService) *PreguntaFrecuenteHandler {
	return &PreguntaFrecuenteHandler{
		service: service,
	}
}

func (h *PreguntaFrecuenteHandler) Listar(c *gin.Context) {
	preguntas, err := h.service.Listar()
	if err != nil {
		log.Println("Error al obtener preguntas frecuentes:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener las preguntas frecuentes",
		})
		return
	}

	c.JSON(http.StatusOK, preguntas)
}

func (h *PreguntaFrecuenteHandler) ObtenerPorID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id de pregunta frecuente inválido",
		})
		return
	}

	pregunta, err := h.service.ObtenerPorID(id)
	if err != nil {
		if errors.Is(err, service.ErrPreguntaFrecuenteNoEncontrada) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "pregunta frecuente no encontrada",
			})
			return
		}

		log.Println("Error al obtener pregunta frecuente:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error al obtener la pregunta frecuente",
		})
		return
	}

	c.JSON(http.StatusOK, pregunta)
}

