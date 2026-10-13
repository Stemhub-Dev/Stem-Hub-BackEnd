package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

type tableroServiceFalso struct {
	llamado        bool
	codigoUsuario  int64
	codigoProyecto *int64
	agrupacion     string
	err            error
}

func (s *tableroServiceFalso) registrar(codigoUsuario int64, codigoProyecto *int64) {
	s.llamado = true
	s.codigoUsuario = codigoUsuario
	s.codigoProyecto = codigoProyecto
}

func (s *tableroServiceFalso) ObtenerIndicadores(codigoUsuario int64, codigoProyecto *int64) (*dto.TableroIndicadoresResponse, error) {
	s.registrar(codigoUsuario, codigoProyecto)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.TableroIndicadoresResponse{}, nil
}

func (s *tableroServiceFalso) ObtenerVersionesPorProyecto(codigoUsuario int64, codigoProyecto *int64) ([]dto.TableroVersionesPorProyectoResponse, error) {
	s.registrar(codigoUsuario, codigoProyecto)
	return []dto.TableroVersionesPorProyectoResponse{}, s.err
}

func (s *tableroServiceFalso) ObtenerCancionesPorEtapa(codigoUsuario int64, codigoProyecto *int64) ([]dto.TableroCancionesPorEtapaResponse, error) {
	s.registrar(codigoUsuario, codigoProyecto)
	return []dto.TableroCancionesPorEtapaResponse{}, s.err
}

func (s *tableroServiceFalso) ObtenerActividad(codigoUsuario int64, codigoProyecto *int64, agrupacion string) ([]dto.TableroActividadResponse, error) {
	s.registrar(codigoUsuario, codigoProyecto)
	s.agrupacion = agrupacion
	return []dto.TableroActividadResponse{}, s.err
}

var rutasTablero = []string{
	"/tablero/indicadores",
	"/tablero/graficos/versiones-por-proyecto",
	"/tablero/graficos/canciones-por-etapa",
	"/tablero/graficos/actividad",
}

func ejecutarTablero(t *testing.T, servicio *tableroServiceFalso, usuario *model.Usuario, url string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h := NewTableroHandler(servicio)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if usuario != nil {
			c.Set(middleware.UsuarioContextKey, usuario)
		}
		c.Next()
	})
	router.GET("/tablero/indicadores", h.ObtenerIndicadores)
	router.GET("/tablero/graficos/versiones-por-proyecto", h.ObtenerVersionesPorProyecto)
	router.GET("/tablero/graficos/canciones-por-etapa", h.ObtenerCancionesPorEtapa)
	router.GET("/tablero/graficos/actividad", h.ObtenerActividad)

	grabador := httptest.NewRecorder()
	router.ServeHTTP(grabador, httptest.NewRequest(http.MethodGet, url, nil))
	return grabador
}

func TestTableroHandler_SinParametrosDevuelve200(t *testing.T) {
	for _, ruta := range rutasTablero {
		t.Run(ruta, func(t *testing.T) {
			servicio := &tableroServiceFalso{}
			grabador := ejecutarTablero(t, servicio, &model.Usuario{CodigoUsuario: 7}, ruta)

			if grabador.Code != http.StatusOK {
				t.Fatalf("status = %d, se esperaba 200", grabador.Code)
			}
			if servicio.codigoUsuario != 7 || servicio.codigoProyecto != nil {
				t.Errorf("parámetros inesperados: usuario %d, proyecto %v", servicio.codigoUsuario, servicio.codigoProyecto)
			}
			// Los gráficos sin datos responden un arreglo vacío, no null.
			if ruta != "/tablero/indicadores" && strings.TrimSpace(grabador.Body.String()) != "[]" {
				t.Errorf("cuerpo = %s, se esperaba []", grabador.Body.String())
			}
		})
	}
}

func TestTableroHandler_PasaProyectoId(t *testing.T) {
	for _, ruta := range rutasTablero {
		t.Run(ruta, func(t *testing.T) {
			servicio := &tableroServiceFalso{}
			grabador := ejecutarTablero(t, servicio, &model.Usuario{CodigoUsuario: 7}, ruta+"?proyectoId=12")

			if grabador.Code != http.StatusOK {
				t.Fatalf("status = %d, se esperaba 200", grabador.Code)
			}
			if servicio.codigoProyecto == nil || *servicio.codigoProyecto != 12 {
				t.Errorf("proyecto = %v, se esperaba 12", servicio.codigoProyecto)
			}
		})
	}
}

func TestTableroHandler_ProyectoIdInvalidoEs400(t *testing.T) {
	for _, valor := range []string{"abc", "0", "-3", "1.5"} {
		for _, ruta := range rutasTablero {
			t.Run(ruta+"?proyectoId="+valor, func(t *testing.T) {
				servicio := &tableroServiceFalso{}
				grabador := ejecutarTablero(t, servicio, &model.Usuario{CodigoUsuario: 7}, ruta+"?proyectoId="+valor)

				if grabador.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, se esperaba 400", grabador.Code)
				}
				if servicio.llamado {
					t.Errorf("no debería llamar al service con un proyectoId inválido")
				}
			})
		}
	}
}

func TestTableroHandler_SinUsuarioEs401(t *testing.T) {
	for _, ruta := range rutasTablero {
		t.Run(ruta, func(t *testing.T) {
			grabador := ejecutarTablero(t, &tableroServiceFalso{}, nil, ruta)

			if grabador.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, se esperaba 401", grabador.Code)
			}
		})
	}
}

func TestTableroHandler_MapeaErrores(t *testing.T) {
	casos := []struct {
		err    error
		status int
	}{
		{service.ErrTableroProyectoNoEncontrado, http.StatusNotFound},
		{service.ErrTableroSinAcceso, http.StatusForbidden},
		{service.ErrTableroAgrupacionInvalida, http.StatusBadRequest},
		{errors.New("detalle interno de la base"), http.StatusInternalServerError},
	}

	for _, caso := range casos {
		for _, ruta := range rutasTablero {
			t.Run(ruta+" "+caso.err.Error(), func(t *testing.T) {
				grabador := ejecutarTablero(t, &tableroServiceFalso{err: caso.err}, &model.Usuario{CodigoUsuario: 7}, ruta+"?proyectoId=3")

				if grabador.Code != caso.status {
					t.Fatalf("status = %d, se esperaba %d", grabador.Code, caso.status)
				}
				if strings.Contains(grabador.Body.String(), "detalle interno") {
					t.Errorf("la respuesta expone el error interno: %s", grabador.Body.String())
				}
			})
		}
	}
}

func TestTableroHandler_AgrupacionPorDefectoEsSemanal(t *testing.T) {
	casos := map[string]string{
		"/tablero/graficos/actividad":                    service.AgrupacionSemanal,
		"/tablero/graficos/actividad?agrupacion=mensual": service.AgrupacionMensual,
		"/tablero/graficos/actividad?agrupacion=semanal": service.AgrupacionSemanal,
		"/tablero/graficos/actividad?agrupacion=anual":   "anual",
	}

	for url, esperado := range casos {
		t.Run(url, func(t *testing.T) {
			servicio := &tableroServiceFalso{}
			ejecutarTablero(t, servicio, &model.Usuario{CodigoUsuario: 7}, url)

			if servicio.agrupacion != esperado {
				t.Errorf("agrupacion = %q, se esperaba %q", servicio.agrupacion, esperado)
			}
		})
	}
}
