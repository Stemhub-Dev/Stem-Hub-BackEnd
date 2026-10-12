package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// comentarioServiceFalso implementa solo Eliminar y EliminarRespuesta; el
// resto de la interfaz queda en el embebido nil.
type comentarioServiceFalso struct {
	service.ComentarioService

	llamado bool
	args    []int64
	err     error
}

func (s *comentarioServiceFalso) Eliminar(
	codigoUsuario, codigoProyecto, codigoCancion, codigoVersion, codigoComentario int64,
) error {
	s.llamado = true
	s.args = []int64{codigoUsuario, codigoProyecto, codigoCancion, codigoVersion, codigoComentario}
	return s.err
}

func (s *comentarioServiceFalso) EliminarRespuesta(
	codigoUsuario, codigoProyecto, codigoCancion, codigoVersion, codigoComentario, codigoRespuesta int64,
) error {
	s.llamado = true
	s.args = []int64{codigoUsuario, codigoProyecto, codigoCancion, codigoVersion, codigoComentario, codigoRespuesta}
	return s.err
}

const (
	rutaEliminarComentario = "/proyectos/:proyectoId/canciones/:cancionId/versiones/:versionId/comentarios/:comentarioId"
	rutaEliminarRespuesta  = rutaEliminarComentario + "/respuestas/:respuestaId"
)

func ejecutarEliminacion(
	t *testing.T,
	servicio *comentarioServiceFalso,
	usuario *model.Usuario,
	url string,
) *httptest.ResponseRecorder {

	t.Helper()

	gin.SetMode(gin.TestMode)

	handler := NewComentarioHandler(servicio)
	autenticar := func(c *gin.Context) {
		if usuario != nil {
			c.Set(middleware.UsuarioContextKey, usuario)
		}
		c.Next()
	}

	router := gin.New()
	router.DELETE(rutaEliminarComentario, autenticar, handler.Eliminar)
	router.DELETE(rutaEliminarRespuesta, autenticar, handler.EliminarRespuesta)

	grabador := httptest.NewRecorder()
	router.ServeHTTP(grabador, httptest.NewRequest(http.MethodDelete, url, nil))

	return grabador
}

func mensajeDeError(t *testing.T, grabador *httptest.ResponseRecorder) string {
	t.Helper()

	var cuerpo map[string]string
	if err := json.Unmarshal(grabador.Body.Bytes(), &cuerpo); err != nil {
		t.Fatalf("cuerpo no es JSON: %q", grabador.Body.String())
	}
	return cuerpo["error"]
}

var usuarioDePrueba = &model.Usuario{CodigoUsuario: 1}

func TestEliminarRespuesta_Exitosa204ConLosIdsDeLaRuta(t *testing.T) {
	servicio := &comentarioServiceFalso{}

	grabador := ejecutarEliminacion(t, servicio, usuarioDePrueba,
		"/proyectos/7/canciones/8/versiones/9/comentarios/10/respuestas/20")

	if grabador.Code != http.StatusNoContent {
		t.Fatalf("status = %d, se esperaba 204", grabador.Code)
	}
	esperados := []int64{1, 7, 8, 9, 10, 20}
	for i, v := range esperados {
		if servicio.args[i] != v {
			t.Fatalf("args = %v, se esperaba %v", servicio.args, esperados)
		}
	}
}

func TestEliminar_ParametrosYUsuarioInvalidos(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		usuario *model.Usuario
		url     string
		status  int
		mensaje string
	}{
		{"proyecto", usuarioDePrueba, "/proyectos/x/canciones/8/versiones/9/comentarios/10", http.StatusBadRequest, "Proyecto inválido"},
		{"canción", usuarioDePrueba, "/proyectos/7/canciones/0/versiones/9/comentarios/10", http.StatusBadRequest, "Canción inválida"},
		{"versión", usuarioDePrueba, "/proyectos/7/canciones/8/versiones/-1/comentarios/10", http.StatusBadRequest, "Versión inválida"},
		{"comentario", usuarioDePrueba, "/proyectos/7/canciones/8/versiones/9/comentarios/abc", http.StatusBadRequest, "Comentario inválido"},
		{"respuesta", usuarioDePrueba, "/proyectos/7/canciones/8/versiones/9/comentarios/10/respuestas/0", http.StatusBadRequest, "Respuesta inválida"},
		{"sin usuario", nil, "/proyectos/7/canciones/8/versiones/9/comentarios/10/respuestas/20", http.StatusUnauthorized, "Usuario no autenticado"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio := &comentarioServiceFalso{}

			grabador := ejecutarEliminacion(t, servicio, caso.usuario, caso.url)

			if grabador.Code != caso.status {
				t.Fatalf("status = %d, se esperaba %d", grabador.Code, caso.status)
			}
			if got := mensajeDeError(t, grabador); got != caso.mensaje {
				t.Fatalf("error = %q, se esperaba %q", got, caso.mensaje)
			}
			if servicio.llamado {
				t.Fatal("no se debía llamar al service")
			}
		})
	}
}

func TestEliminarRespuesta_TraduceErroresDelService(t *testing.T) {
	url := "/proyectos/7/canciones/8/versiones/9/comentarios/10/respuestas/20"

	for _, caso := range []struct {
		nombre  string
		err     error
		status  int
		mensaje string
	}{
		{"no existe", service.ErrRespuestaComentarioNoEncontrada, http.StatusNotFound, "El contenido solicitado no existe"},
		{"versión inexistente", service.ErrComentarioVersionNoEncontrada, http.StatusNotFound, "El contenido solicitado no existe"},
		{"ajena sin permiso", service.ErrComentarioSinPermisoEliminar, http.StatusForbidden, "No tenés permisos para eliminar esta respuesta"},
		{"fuera del proyecto", service.ErrComentarioSinAcceso, http.StatusForbidden, "No tenés acceso a este proyecto"},
		{"error interno", errors.New("db caída"), http.StatusInternalServerError, "Error al eliminar la respuesta"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			grabador := ejecutarEliminacion(t, &comentarioServiceFalso{err: caso.err}, usuarioDePrueba, url)

			if grabador.Code != caso.status {
				t.Fatalf("status = %d, se esperaba %d", grabador.Code, caso.status)
			}
			if got := mensajeDeError(t, grabador); got != caso.mensaje {
				t.Fatalf("error = %q, se esperaba %q", got, caso.mensaje)
			}
		})
	}
}

// El refactor de Eliminar a los helpers compartidos mantiene sus respuestas.
func TestEliminarComentario_MantieneSusRespuestas(t *testing.T) {
	url := "/proyectos/7/canciones/8/versiones/9/comentarios/10"

	for _, caso := range []struct {
		nombre  string
		err     error
		status  int
		mensaje string
	}{
		{"no existe", service.ErrComentarioNoEncontrado, http.StatusNotFound, "El contenido solicitado no existe"},
		{"ajeno sin permiso", service.ErrComentarioSinPermisoEliminar, http.StatusForbidden, "No tenés permisos para eliminar este comentario"},
		{"error interno", errors.New("db caída"), http.StatusInternalServerError, "Error al eliminar el comentario"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			grabador := ejecutarEliminacion(t, &comentarioServiceFalso{err: caso.err}, usuarioDePrueba, url)

			if grabador.Code != caso.status {
				t.Fatalf("status = %d, se esperaba %d", grabador.Code, caso.status)
			}
			if got := mensajeDeError(t, grabador); got != caso.mensaje {
				t.Fatalf("error = %q, se esperaba %q", got, caso.mensaje)
			}
		})
	}

	servicio := &comentarioServiceFalso{}
	if grabador := ejecutarEliminacion(t, servicio, usuarioDePrueba, url); grabador.Code != http.StatusNoContent {
		t.Fatalf("status = %d, se esperaba 204", grabador.Code)
	}
	if len(servicio.args) != 5 || servicio.args[4] != 10 {
		t.Fatalf("args = %v", servicio.args)
	}
}
