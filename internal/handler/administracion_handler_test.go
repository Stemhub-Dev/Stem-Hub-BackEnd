package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// HU-SEG-F05: el handler pasa el admin que actúa y traduce los rechazos a 409.

type administracionRepositoryFalso struct {
	repository.UsuarioAdministracionRepository
	err error
}

func (r administracionRepositoryFalso) CambiarEstado(codigo int64, activo bool) (*dto.CambiarEstadoUsuarioResponse, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &dto.CambiarEstadoUsuarioResponse{CodigoUsuario: codigo, Activo: activo}, nil
}

func (r administracionRepositoryFalso) ActualizarAdministracion(codigo int64, admin bool, activo bool) (*dto.ActualizarAdministracionUsuarioResponse, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &dto.ActualizarAdministracionUsuarioResponse{CodigoUsuario: codigo, EsAdministradorSistema: admin, Activo: activo}, nil
}

func ejecutarAdministracion(t *testing.T, repo administracionRepositoryFalso, metodo, ruta, cuerpo string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h := NewUsuarioAdministracionHandler(service.NewUsuarioAdministracionService(repo))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.UsuarioContextKey, &model.Usuario{CodigoUsuario: 10})
		c.Next()
	})
	router.PATCH("/usuarios/:id/estado", h.CambiarEstado)
	router.PATCH("/usuarios/:id/administracion", h.ActualizarAdministracion)

	grabador := httptest.NewRecorder()
	request := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(grabador, request)
	return grabador
}

func TestAdministracion_RechazosResponden409(t *testing.T) {
	casos := []struct {
		nombre string
		repo   administracionRepositoryFalso
		ruta   string
		cuerpo string
		quiero string
	}{
		{"darse de baja", administracionRepositoryFalso{}, "/usuarios/10/estado", `{"activo": false}`, "No podés darte de baja a vos mismo."},
		{"quitarse el rol", administracionRepositoryFalso{}, "/usuarios/10/administracion", `{"esAdministradorSistema": false, "activo": true}`, "No podés quitarte el rol Administrador."},
		{"último admin", administracionRepositoryFalso{err: repository.ErrSinAdministradoresActivos}, "/usuarios/20/estado", `{"activo": false}`, "Tiene que quedar al menos un Administrador activo."},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			grabador := ejecutarAdministracion(t, caso.repo, http.MethodPatch, caso.ruta, caso.cuerpo)

			if grabador.Code != http.StatusConflict || !strings.Contains(grabador.Body.String(), caso.quiero) {
				t.Fatalf("respuesta = %d %s, quería 409 %q", grabador.Code, grabador.Body.String(), caso.quiero)
			}
		})
	}
}

func TestAdministracion_DarDeBajaAOtroUsuario(t *testing.T) {
	grabador := ejecutarAdministracion(t, administracionRepositoryFalso{}, http.MethodPatch, "/usuarios/20/estado", `{"activo": false}`)

	if grabador.Code != http.StatusOK {
		t.Fatalf("status = %d, quería 200 (body %s)", grabador.Code, grabador.Body.String())
	}
}
