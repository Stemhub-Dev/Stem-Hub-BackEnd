package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

// Tests de integración de HU-COM-B05 (responder un comentario) y HU-COM-B07
// (cambiar el estado de un comentario) sobre el escenario de versiones.

type escenarioComentarios struct {
	*escenarioVersiones
	comentario int64
}

func prepararEscenarioComentarios(t *testing.T) escenarioComentarios {
	t.Helper()
	e := prepararEscenarioVersiones(t)

	h := handler.NewComentarioHandler(service.NewComentarioService(
		repository.NewComentarioRepository(e.db),
		repository.NewProyectoRepository(e.db),
		repository.NewCancionRepository(e.db),
		repository.NewIntegranteRepository(e.db),
		nil,
		storageDePrueba{},
	))
	base := "/proyectos/:proyectoId/canciones/:cancionId/versiones/:versionId/comentarios"
	e.router.POST(base, h.Crear)
	e.router.POST(base+"/:comentarioId/respuestas", h.Responder)
	e.router.PATCH(base+"/:comentarioId/estado", h.CambiarEstado)

	t.Cleanup(func() {
		_, _ = e.db.Exec(`DELETE FROM comentariorespuesta WHERE codigocomentario IN
			(SELECT codigocomentario FROM comentario WHERE codigocancionversion = ANY($1))`, e.versiones)
		_, _ = e.db.Exec(`DELETE FROM comentario WHERE codigocancionversion = ANY($1)`, e.versiones)
	})

	ec := escenarioComentarios{escenarioVersiones: e}
	code, body := e.pedir(t, e.productor, http.MethodPost, ec.rutaComentarios(), map[string]any{"texto": "Subí el bajo"})
	esperarStatus(t, code, body, http.StatusCreated, "")
	id, ok := body["codigoComentario"].(float64)
	if !ok {
		t.Fatalf("alta del comentario: %v", body)
	}
	ec.comentario = int64(id)
	return ec
}

func (e escenarioComentarios) rutaComentarios() string {
	return e.ruta(e.versiones[2]) + "/comentarios"
}

func (e escenarioComentarios) rutaComentario(sufijo string) string {
	return fmt.Sprintf("%s/%d/%s", e.rutaComentarios(), e.comentario, sufijo)
}

// HU-COM-B05

func TestRespuestas_CrearDevuelveLaRespuesta(t *testing.T) {
	e := prepararEscenarioComentarios(t)

	// Cualquier integrante responde, también un Músico.
	code, body := e.pedir(t, e.musico, http.MethodPost, e.rutaComentario("respuestas"),
		map[string]any{"texto": "  Dale, lo subo  "})
	esperarStatus(t, code, body, http.StatusCreated, "")

	if body["codigoRespuesta"] == nil || body["texto"] != "Dale, lo subo" || body["esPropia"] != true {
		t.Fatalf("respuesta = %v", body)
	}
	if body["fechaHoraAlta"] == nil {
		t.Fatalf("la respuesta no trae fecha: %v", body)
	}
	autor, _ := body["autor"].(map[string]any)
	if autor == nil || autor["nombre"] != "Ver musico" {
		t.Fatalf("autor = %v", body["autor"])
	}
}

func TestRespuestas_Errores(t *testing.T) {
	e := prepararEscenarioComentarios(t)
	ajeno := e.nuevoAjeno(t)

	casos := []struct {
		nombre  string
		usuario int64
		ruta    string
		texto   string
		status  int
		mensaje string
	}{
		{"vacía", e.productor, e.rutaComentario("respuestas"), "", http.StatusBadRequest,
			"La respuesta no puede estar vacía"},
		{"solo espacios", e.productor, e.rutaComentario("respuestas"), "   ", http.StatusBadRequest,
			"La respuesta no puede estar vacía"},
		{"más de 200 caracteres", e.productor, e.rutaComentario("respuestas"), strings.Repeat("ñ", 201),
			http.StatusBadRequest, "La respuesta no puede superar los 200 caracteres"},
		{"comentario inexistente", e.productor, e.rutaComentarios() + "/999999999/respuestas", "Hola",
			http.StatusNotFound, ""},
		{"usuario ajeno al proyecto", ajeno, e.rutaComentario("respuestas"), "Hola",
			http.StatusForbidden, ""},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			code, body := e.pedir(t, caso.usuario, http.MethodPost, caso.ruta, map[string]any{"texto": caso.texto})
			esperarStatus(t, code, body, caso.status, caso.mensaje)
		})
	}
}

// Mismo criterio para el comentario (HU-COM-B02): antes el binding del DTO
// cortaba con "Solicitud inválida" en vez de estos mensajes.
func TestComentarios_TextoVacioOLargoDevuelveElMensajeDelCaso(t *testing.T) {
	e := prepararEscenarioComentarios(t)

	code, body := e.pedir(t, e.productor, http.MethodPost, e.rutaComentarios(), map[string]any{"texto": ""})
	esperarStatus(t, code, body, http.StatusBadRequest, "El comentario no puede estar vacío")

	code, body = e.pedir(t, e.productor, http.MethodPost, e.rutaComentarios(),
		map[string]any{"texto": strings.Repeat("ñ", 201)})
	esperarStatus(t, code, body, http.StatusBadRequest, "El comentario no puede superar los 200 caracteres")
}

func TestRespuestas_ExactamenteDoscientosCaracteresSeAcepta(t *testing.T) {
	e := prepararEscenarioComentarios(t)

	code, body := e.pedir(t, e.productor, http.MethodPost, e.rutaComentario("respuestas"),
		map[string]any{"texto": strings.Repeat("ñ", 200)})

	esperarStatus(t, code, body, http.StatusCreated, "")
}

// HU-COM-B07

func TestEstadoComentario_PendienteYResuelto(t *testing.T) {
	e := prepararEscenarioComentarios(t)

	// Cualquier integrante puede cambiarlo, no solo quien lo escribió.
	for _, estado := range []string{"Resuelto", "Pendiente"} {
		code, body := e.pedir(t, e.musico, http.MethodPatch, e.rutaComentario("estado"),
			map[string]any{"estado": estado})
		esperarStatus(t, code, body, http.StatusOK, "")
		if body["estado"] != estado || body["codigoComentario"] != float64(e.comentario) {
			t.Fatalf("respuesta al pasar a %s = %v", estado, body)
		}
	}
}

func TestEstadoComentario_Errores(t *testing.T) {
	e := prepararEscenarioComentarios(t)
	ajeno := e.nuevoAjeno(t)

	casos := []struct {
		nombre  string
		usuario int64
		ruta    string
		estado  string
		status  int
		mensaje string
	}{
		{"estado inválido", e.productor, e.rutaComentario("estado"), "Archivado", http.StatusBadRequest,
			"El estado debe ser Pendiente o Resuelto"},
		{"comentario inexistente", e.productor, e.rutaComentarios() + "/999999999/estado", "Resuelto",
			http.StatusNotFound, ""},
		{"usuario ajeno al proyecto", ajeno, e.rutaComentario("estado"), "Resuelto",
			http.StatusForbidden, ""},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			code, body := e.pedir(t, caso.usuario, http.MethodPatch, caso.ruta, map[string]any{"estado": caso.estado})
			esperarStatus(t, code, body, caso.status, caso.mensaje)
		})
	}
}
