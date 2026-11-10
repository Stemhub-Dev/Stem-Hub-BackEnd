package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

const nombreDescargaManual = "manual-usuario-stemhub.pdf"

type AyudaHandler struct {
	service service.AyudaService
}

func NewAyudaHandler(service service.AyudaService) *AyudaHandler {
	return &AyudaHandler{service: service}
}

func (h *AyudaHandler) ListarPreguntasFrecuentes(c *gin.Context) {
	preguntas, err := h.service.ListarPreguntasFrecuentes()
	if err != nil {
		log.Println("Error al obtener preguntas frecuentes:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener las preguntas frecuentes"})
		return
	}

	c.JSON(http.StatusOK, preguntas)
}

func (h *AyudaHandler) DescargarManual(c *gin.Context) {
	ruta, err := h.service.ObtenerRutaManual()

	if errors.Is(err, service.ErrManualUsuarioNoDisponible) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		log.Println("Error al obtener el manual de usuario:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener el manual de usuario"})
		return
	}

	c.FileAttachment(ruta, nombreDescargaManual)
}
