package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// HU-PER-B01: GET y PUT /perfil.

type perfilServiceFalso struct {
	err error

	nombre      string
	descripcion *string
	avatar      *service.ArchivoImagen
}

func (s *perfilServiceFalso) ObtenerPerfil(int64) (*model.Integrante, *string, error) {
	url := "https://storage.test/integrantes/3/avatar.png"
	return &model.Integrante{CodIntegrante: 3, NombreIntegrante: "Ana", DescripcionIntegrante: textoPerfil("Bajista")}, &url, s.err
}

func (s *perfilServiceFalso) EditarPerfil(_ int64, nombre string, descripcion *string, avatar *service.ArchivoImagen, _ bool) (*model.Integrante, *string, error) {
	s.nombre, s.descripcion, s.avatar = nombre, descripcion, avatar
	if s.err != nil {
		return nil, nil, s.err
	}
	return &model.Integrante{CodIntegrante: 3, NombreIntegrante: strings.TrimSpace(nombre), DescripcionIntegrante: descripcion}, nil, nil
}

type rolesFalsos struct{}

func (rolesFalsos) AsignarRol(int64, int64) error              { return nil }
func (rolesFalsos) EsAdministradorSistema(int64) (bool, error) { return false, nil }

func textoPerfil(valor string) *string { return &valor }

func ejecutarPerfil(t *testing.T, servicio *perfilServiceFalso, request *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h := NewIntegranteHandler(servicio, rolesFalsos{})
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.UsuarioContextKey, &model.Usuario{CodigoUsuario: 7, Email: "ana@test.com"})
		c.Next()
	})
	router.GET("/perfil", h.ObtenerPerfil)
	router.PUT("/perfil", h.EditarPerfil)

	grabador := httptest.NewRecorder()
	router.ServeHTTP(grabador, request)
	return grabador
}

func pedidoPerfil(t *testing.T, campos map[string]string, avatar string) *http.Request {
	t.Helper()

	var cuerpo bytes.Buffer
	escritor := multipart.NewWriter(&cuerpo)
	for campo, valor := range campos {
		if err := escritor.WriteField(campo, valor); err != nil {
			t.Fatal(err)
		}
	}
	if avatar != "" {
		parte, err := escritor.CreateFormFile("avatar", avatar)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.WriteString(parte, "imagen")
	}
	_ = escritor.Close()

	request := httptest.NewRequest(http.MethodPut, "/perfil", &cuerpo)
	request.Header.Set("Content-Type", escritor.FormDataContentType())
	return request
}

func TestObtenerPerfil_DevuelveNombreDescripcionYAvatar(t *testing.T) {
	grabador := ejecutarPerfil(t, &perfilServiceFalso{}, httptest.NewRequest(http.MethodGet, "/perfil", nil))

	if grabador.Code != http.StatusOK {
		t.Fatalf("status = %d, quería 200", grabador.Code)
	}
	var perfil map[string]any
	if err := json.Unmarshal(grabador.Body.Bytes(), &perfil); err != nil {
		t.Fatal(err)
	}
	if perfil["nombre"] != "Ana" || perfil["descripcion"] != "Bajista" || perfil["avatarUrl"] != "https://storage.test/integrantes/3/avatar.png" {
		t.Fatalf("perfil = %v", perfil)
	}
}

func TestEditarPerfil_MultipartConAvatar(t *testing.T) {
	servicio := &perfilServiceFalso{}

	grabador := ejecutarPerfil(t, servicio, pedidoPerfil(t, map[string]string{"nombre": "Ana", "descripcion": "Bajista"}, "foto.png"))

	if grabador.Code != http.StatusOK {
		t.Fatalf("status = %d, quería 200 (%s)", grabador.Code, grabador.Body.String())
	}
	if servicio.nombre != "Ana" || servicio.descripcion == nil || *servicio.descripcion != "Bajista" {
		t.Fatalf("recibió %q / %v", servicio.nombre, servicio.descripcion)
	}
	if servicio.avatar == nil || servicio.avatar.NombreOriginal != "foto.png" {
		t.Fatalf("avatar = %+v", servicio.avatar)
	}
}

func TestEditarPerfil_MensajesDeError(t *testing.T) {
	casos := []struct {
		err    error
		status int
		quiero string
	}{
		{service.ErrPerfilNombreObligatorio, http.StatusBadRequest, "El nombre no puede estar vacío"},
		{service.ErrPerfilAvatarFormatoInvalido, http.StatusBadRequest, "El formato de la imagen no está soportado"},
		{service.ErrPerfilAvatarDemasiadoGrande, http.StatusBadRequest, "La imagen supera el tamaño máximo permitido"},
		{service.ErrPerfilErrorAlmacenamiento, http.StatusInternalServerError, "Error al almacenar el avatar"},
		{errors.New("base caída"), http.StatusInternalServerError, "Error al editar el perfil"},
	}

	for _, caso := range casos {
		t.Run(caso.quiero, func(t *testing.T) {
			grabador := ejecutarPerfil(t, &perfilServiceFalso{err: caso.err}, pedidoPerfil(t, map[string]string{"nombre": "Ana"}, ""))

			if grabador.Code != caso.status || !strings.Contains(grabador.Body.String(), caso.quiero) {
				t.Fatalf("respuesta = %d %s, quería %d %q", grabador.Code, grabador.Body.String(), caso.status, caso.quiero)
			}
		})
	}
}
