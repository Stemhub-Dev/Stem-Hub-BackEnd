package handler

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// portadaServiceFalso registra lo que llega al alta y a la modificación.
type portadaServiceFalso struct {
	service.ProyectoService

	crearLlamado  bool
	pedidoAlta    dto.CrearProyectoRequest
	pedidoEdicion dto.EditarProyectoRequest
	nombreLogo    string
	contenidoLogo string
	eliminarLogo  bool

	err error
}

func (s *portadaServiceFalso) registrarLogo(logo *service.ArchivoImagen) {
	if logo == nil {
		return
	}
	s.nombreLogo = logo.NombreOriginal
	contenido, _ := io.ReadAll(logo.Contenido)
	s.contenidoLogo = string(contenido)
}

func (s *portadaServiceFalso) CrearProyecto(
	_ int64,
	request dto.CrearProyectoRequest,
	logo *service.ArchivoImagen,
) (*dto.CrearProyectoResponse, error) {
	s.crearLlamado = true
	s.pedidoAlta = request
	s.registrarLogo(logo)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.CrearProyectoResponse{CodigoProyecto: 9}, nil
}

func (s *portadaServiceFalso) EditarProyecto(
	_ int64,
	_ int64,
	request dto.EditarProyectoRequest,
	logo *service.ArchivoImagen,
	eliminarLogo bool,
) (*dto.EditarProyectoResponse, error) {
	s.pedidoEdicion = request
	s.registrarLogo(logo)
	s.eliminarLogo = eliminarLogo
	if s.err != nil {
		return nil, s.err
	}
	return &dto.EditarProyectoResponse{CodigoProyecto: 9}, nil
}

type campoForm struct {
	nombre string
	valor  string
}

func formularioMultipart(t *testing.T, campos []campoForm, logo string) (*bytes.Buffer, string) {
	t.Helper()

	cuerpo := &bytes.Buffer{}
	escritor := multipart.NewWriter(cuerpo)

	for _, campo := range campos {
		if err := escritor.WriteField(campo.nombre, campo.valor); err != nil {
			t.Fatal(err)
		}
	}

	if logo != "" {
		parte, err := escritor.CreateFormFile("logo", logo)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parte.Write([]byte("bytes de la imagen")); err != nil {
			t.Fatal(err)
		}
	}

	if err := escritor.Close(); err != nil {
		t.Fatal(err)
	}

	return cuerpo, escritor.FormDataContentType()
}

func camposAltaCompletos() []campoForm {
	return []campoForm{
		{"nombre", "Rock Nacional"},
		{"descripcion", "Primer disco"},
		{"codigoTipoProyecto", "1"},
		{"codigosGeneros", "3"},
		{"codigosGeneros", "5"},
		{"codRol", "2"},
	}
}

func ejecutarPortada(
	t *testing.T,
	servicio *portadaServiceFalso,
	metodo string,
	url string,
	cuerpo io.Reader,
	contentType string,
) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	handler := NewProyectoHandler(servicio)
	router := gin.New()
	conUsuario := func(c *gin.Context) {
		c.Set(middleware.UsuarioContextKey, &model.Usuario{CodigoUsuario: 1})
		c.Next()
	}
	router.POST("/proyectos/crear", conUsuario, handler.Crear)
	router.PATCH("/proyectos/:proyectoId", conUsuario, handler.Editar)

	pedido := httptest.NewRequest(metodo, url, cuerpo)
	pedido.Header.Set("Content-Type", contentType)

	grabador := httptest.NewRecorder()
	router.ServeHTTP(grabador, pedido)

	return grabador
}

func TestCrearProyecto_MultipartConPortada(t *testing.T) {
	servicio := &portadaServiceFalso{}
	cuerpo, contentType := formularioMultipart(t, camposAltaCompletos(), "tapa.png")

	grabador := ejecutarPortada(t, servicio, http.MethodPost, "/proyectos/crear", cuerpo, contentType)

	if grabador.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", grabador.Code, grabador.Body.String())
	}

	descripcion := "Primer disco"
	esperado := dto.CrearProyectoRequest{
		Nombre:             "Rock Nacional",
		Descripcion:        &descripcion,
		CodigoTipoProyecto: 1,
		CodigosGeneros:     []int64{3, 5},
		CodRol:             2,
	}
	if !reflect.DeepEqual(servicio.pedidoAlta, esperado) {
		t.Fatalf("pedido = %+v", servicio.pedidoAlta)
	}
	if servicio.nombreLogo != "tapa.png" || servicio.contenidoLogo != "bytes de la imagen" {
		t.Fatalf("logo = %q (%q)", servicio.nombreLogo, servicio.contenidoLogo)
	}
}

func TestCrearProyecto_MultipartSinPortada(t *testing.T) {
	servicio := &portadaServiceFalso{}
	cuerpo, contentType := formularioMultipart(t, camposAltaCompletos(), "")

	grabador := ejecutarPortada(t, servicio, http.MethodPost, "/proyectos/crear", cuerpo, contentType)

	if grabador.Code != http.StatusCreated || servicio.nombreLogo != "" {
		t.Fatalf("status = %d, logo = %q", grabador.Code, servicio.nombreLogo)
	}
}

func TestCrearProyecto_JSONSigueFuncionando(t *testing.T) {
	servicio := &portadaServiceFalso{}
	cuerpo := strings.NewReader(`{"nombre":"Jazz","codigoTipoProyecto":1,"codigosGeneros":[3],"codRol":2}`)

	grabador := ejecutarPortada(t, servicio, http.MethodPost, "/proyectos/crear", cuerpo, "application/json")

	if grabador.Code != http.StatusCreated || servicio.pedidoAlta.Nombre != "Jazz" || servicio.nombreLogo != "" {
		t.Fatalf("status = %d, pedido = %+v", grabador.Code, servicio.pedidoAlta)
	}
}

func TestCrearProyecto_MultipartIncompletoResponde400(t *testing.T) {
	sin := func(campo string) []campoForm {
		var campos []campoForm
		for _, c := range camposAltaCompletos() {
			if c.nombre != campo {
				campos = append(campos, c)
			}
		}
		return campos
	}

	casos := map[string][]campoForm{
		"sin nombre":       sin("nombre"),
		"sin tipo":         sin("codigoTipoProyecto"),
		"sin géneros":      sin("codigosGeneros"),
		"sin rol":          sin("codRol"),
		"tipo inválido":    append(sin("codigoTipoProyecto"), campoForm{"codigoTipoProyecto", "abc"}),
		"género inválido":  append(sin("codigosGeneros"), campoForm{"codigosGeneros", "0"}),
		"nombre en blanco": append(sin("nombre"), campoForm{"nombre", "   "}),
	}

	for nombre, campos := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio := &portadaServiceFalso{}
			cuerpo, contentType := formularioMultipart(t, campos, "tapa.png")

			grabador := ejecutarPortada(t, servicio, http.MethodPost, "/proyectos/crear", cuerpo, contentType)

			if grabador.Code != http.StatusBadRequest || servicio.crearLlamado {
				t.Fatalf("status = %d, service llamado = %v", grabador.Code, servicio.crearLlamado)
			}
		})
	}
}

func TestPortadaProyecto_ErroresDeLaImagen(t *testing.T) {
	casos := []struct {
		err     error
		status  int
		mensaje string
	}{
		{service.ErrProyectoLogoFormatoInvalido, http.StatusBadRequest, "El formato de la imagen no está soportado"},
		{service.ErrProyectoLogoDemasiadoGrande, http.StatusBadRequest, "La imagen supera el tamaño máximo permitido"},
		{service.ErrProyectoErrorAlmacenamiento, http.StatusInternalServerError, "Error al almacenar el logo del proyecto"},
	}

	for _, caso := range casos {
		t.Run("alta: "+caso.mensaje, func(t *testing.T) {
			servicio := &portadaServiceFalso{err: caso.err}
			cuerpo, contentType := formularioMultipart(t, camposAltaCompletos(), "tapa.png")

			grabador := ejecutarPortada(t, servicio, http.MethodPost, "/proyectos/crear", cuerpo, contentType)

			if grabador.Code != caso.status || mensajeDeError(t, grabador) != caso.mensaje {
				t.Fatalf("status = %d, cuerpo = %s", grabador.Code, grabador.Body.String())
			}
		})

		t.Run("edición: "+caso.mensaje, func(t *testing.T) {
			servicio := &portadaServiceFalso{err: caso.err}
			cuerpo, contentType := formularioMultipart(t, nil, "tapa.png")

			grabador := ejecutarPortada(t, servicio, http.MethodPatch, "/proyectos/9", cuerpo, contentType)

			if grabador.Code != caso.status || mensajeDeError(t, grabador) != caso.mensaje {
				t.Fatalf("status = %d, cuerpo = %s", grabador.Code, grabador.Body.String())
			}
		})
	}
}

func TestEditarProyecto_MultipartPasaPortadaYPedidoDeQuitar(t *testing.T) {
	casos := []struct {
		nombre       string
		campos       []campoForm
		logo         string
		eliminarLogo bool
	}{
		{"portada nueva", []campoForm{{"nombre", "Rock"}}, "tapa.webp", false},
		{"quitar portada", []campoForm{{"eliminarLogo", "true"}}, "", true},
		{"sin tocar la portada", []campoForm{{"nombre", "Rock"}}, "", false},
		{"eliminarLogo distinto de true", []campoForm{{"eliminarLogo", "si"}}, "", false},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio := &portadaServiceFalso{}
			cuerpo, contentType := formularioMultipart(t, caso.campos, caso.logo)

			grabador := ejecutarPortada(t, servicio, http.MethodPatch, "/proyectos/9", cuerpo, contentType)

			if grabador.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", grabador.Code, grabador.Body.String())
			}
			if servicio.nombreLogo != caso.logo || servicio.eliminarLogo != caso.eliminarLogo {
				t.Fatalf("logo = %q, eliminarLogo = %v", servicio.nombreLogo, servicio.eliminarLogo)
			}
		})
	}
}

func TestEditarProyecto_JSONNoQuitaLaPortada(t *testing.T) {
	servicio := &portadaServiceFalso{}
	cuerpo := strings.NewReader(`{"nombre":"Rock","eliminarLogo":"true"}`)

	grabador := ejecutarPortada(t, servicio, http.MethodPatch, "/proyectos/9", cuerpo, "application/json")

	if grabador.Code != http.StatusOK || servicio.eliminarLogo {
		t.Fatalf("status = %d, eliminarLogo = %v", grabador.Code, servicio.eliminarLogo)
	}
}
