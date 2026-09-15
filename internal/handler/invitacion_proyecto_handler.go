package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type InvitacionProyectoHandler struct {
	service service.InvitacionProyectoService
}

func NewInvitacionProyectoHandler(
	service service.InvitacionProyectoService,
) *InvitacionProyectoHandler {

	return &InvitacionProyectoHandler{
		service: service,
	}
}

func usuarioAutenticado(c *gin.Context) (*model.Usuario, bool) {

	valorUsuario, existe := c.Get(middleware.UsuarioContextKey)

	if !existe {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return nil, false
	}

	usuario, ok := valorUsuario.(*model.Usuario)

	if !ok || usuario == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return nil, false
	}

	return usuario, true
}

func (h *InvitacionProyectoHandler) Crear(c *gin.Context) {

	usuario, ok := usuarioAutenticado(c)

	if !ok {
		return
	}

	codigoProyecto, err := strconv.ParseInt(c.Param("proyectoId"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Proyecto inválido"})
		return
	}

	var request dto.CrearInvitacionRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Solicitud inválida"})
		return
	}

	invitacion, err := h.service.Invitar(usuario.CodigoUsuario, codigoProyecto, request)

	switch {

	case errors.Is(err, service.ErrProyectoNoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"error": "El proyecto no existe"})

	case errors.Is(err, service.ErrInvitacionNoEsOwner):
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el propietario del proyecto puede invitar"})

	case errors.Is(err, service.ErrInvitacionEmailInvalido):
		c.JSON(http.StatusBadRequest, gin.H{"error": "El email es inválido"})

	case errors.Is(err, service.ErrInvitacionRolNoValido):
		c.JSON(http.StatusBadRequest, gin.H{"error": "El rol seleccionado no es válido para un proyecto"})

	case errors.Is(err, service.ErrInvitacionYaEsIntegrante):
		c.JSON(http.StatusConflict, gin.H{"error": "Ese usuario ya es integrante del proyecto"})

	case err != nil:
		log.Println("Error al crear invitación:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear la invitación"})

	default:
		c.JSON(http.StatusCreated, invitacion)
	}
}

func (h *InvitacionProyectoHandler) ListarPendientes(c *gin.Context) {

	usuario, ok := usuarioAutenticado(c)

	if !ok {
		return
	}

	codigoProyecto, err := strconv.ParseInt(c.Param("proyectoId"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Proyecto inválido"})
		return
	}

	invitaciones, err := h.service.ListarPendientes(usuario.CodigoUsuario, codigoProyecto)

	switch {

	case errors.Is(err, service.ErrProyectoNoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"error": "El proyecto no existe"})

	case errors.Is(err, service.ErrInvitacionNoEsOwner):
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el propietario del proyecto puede ver las invitaciones"})

	case err != nil:
		log.Println("Error al listar invitaciones:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener las invitaciones"})

	default:
		c.JSON(http.StatusOK, invitaciones)
	}
}

func (h *InvitacionProyectoHandler) Cancelar(c *gin.Context) {

	usuario, ok := usuarioAutenticado(c)

	if !ok {
		return
	}

	codigoProyecto, err := strconv.ParseInt(c.Param("proyectoId"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Proyecto inválido"})
		return
	}

	codigoInvitacion, err := strconv.ParseInt(c.Param("invitacionId"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invitación inválida"})
		return
	}

	err = h.service.Cancelar(usuario.CodigoUsuario, codigoProyecto, codigoInvitacion)

	switch {

	case errors.Is(err, service.ErrProyectoNoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"error": "El proyecto no existe"})

	case errors.Is(err, service.ErrInvitacionNoEsOwner):
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo el propietario del proyecto puede cancelar invitaciones"})

	case errors.Is(err, service.ErrInvitacionNoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"error": "La invitación no existe"})

	case err != nil:
		log.Println("Error al cancelar invitación:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al cancelar la invitación"})

	default:
		c.JSON(http.StatusNoContent, nil)
	}
}

func (h *InvitacionProyectoHandler) ObtenerDetalle(c *gin.Context) {

	token := c.Param("token")

	detalle, err := h.service.ObtenerDetalle(token)

	switch {

	case errors.Is(err, service.ErrInvitacionNoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"error": "La invitación no existe"})

	case err != nil:
		log.Println("Error al obtener detalle de invitación:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener la invitación"})

	default:
		c.JSON(http.StatusOK, detalle)
	}
}

func (h *InvitacionProyectoHandler) MisInvitaciones(c *gin.Context) {

	usuario, ok := usuarioAutenticado(c)

	if !ok {
		return
	}

	invitaciones, err := h.service.ListarMisInvitaciones(usuario)

	if err != nil {
		log.Println("Error al listar mis invitaciones:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener las invitaciones"})
		return
	}

	c.JSON(http.StatusOK, invitaciones)
}

func (h *InvitacionProyectoHandler) Rechazar(c *gin.Context) {

	usuario, ok := usuarioAutenticado(c)

	if !ok {
		return
	}

	token := c.Param("token")

	err := h.service.Rechazar(usuario, token)

	switch {

	case errors.Is(err, service.ErrInvitacionNoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"error": "La invitación no existe"})

	case errors.Is(err, service.ErrInvitacionVencida):
		c.JSON(http.StatusGone, gin.H{"error": "La invitación venció"})

	case errors.Is(err, service.ErrInvitacionCancelada):
		c.JSON(http.StatusGone, gin.H{"error": "La invitación fue cancelada"})

	case errors.Is(err, service.ErrInvitacionYaAceptada):
		c.JSON(http.StatusConflict, gin.H{"error": "La invitación ya fue aceptada"})

	case errors.Is(err, service.ErrInvitacionEmailNoCoincide):
		c.JSON(http.StatusForbidden, gin.H{"error": "El email de tu cuenta no coincide con el de la invitación"})

	case err != nil:
		log.Println("Error al rechazar invitación:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al rechazar la invitación"})

	default:
		c.JSON(http.StatusNoContent, nil)
	}
}

func (h *InvitacionProyectoHandler) Aceptar(c *gin.Context) {

	usuario, ok := usuarioAutenticado(c)

	if !ok {
		return
	}

	token := c.Param("token")

	codigoProyecto, err := h.service.Aceptar(usuario, token)

	switch {

	case errors.Is(err, service.ErrInvitacionNoEncontrada):
		c.JSON(http.StatusNotFound, gin.H{"error": "La invitación no existe"})

	case errors.Is(err, service.ErrInvitacionVencida):
		c.JSON(http.StatusGone, gin.H{"error": "La invitación venció"})

	case errors.Is(err, service.ErrInvitacionCancelada):
		c.JSON(http.StatusGone, gin.H{"error": "La invitación fue cancelada"})

	case errors.Is(err, service.ErrInvitacionYaAceptada):
		c.JSON(http.StatusConflict, gin.H{"error": "La invitación ya fue aceptada"})

	case errors.Is(err, service.ErrInvitacionEmailNoCoincide):
		c.JSON(http.StatusForbidden, gin.H{"error": "El email de tu cuenta no coincide con el de la invitación"})

	case err != nil:
		log.Println("Error al aceptar invitación:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al aceptar la invitación"})

	default:
		c.JSON(http.StatusOK, dto.AceptarInvitacionResponse{CodigoProyecto: codigoProyecto})
	}
}
