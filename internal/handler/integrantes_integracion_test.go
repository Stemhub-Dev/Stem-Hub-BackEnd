package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
)

// Tests de integración de HU-VER-B02 (gestión de integrantes) sobre el
// escenario de versiones: el productor es el propietario y el músico, no.

type escenarioIntegrantes struct {
	*escenarioVersiones
	integranteProductor int64
	integranteMusico    int64
}

func prepararEscenarioIntegrantes(t *testing.T) escenarioIntegrantes {
	t.Helper()
	e := prepararEscenarioVersiones(t)

	h := handler.NewProyectoHandler(service.NewProyectoService(
		repository.NewProyectoRepository(e.db),
		repository.NewIntegranteRepository(e.db),
		storageDePrueba{},
	))
	e.router.GET("/proyectos/:proyectoId/integrantes", h.ListarColaboradores)
	e.router.PATCH("/proyectos/:proyectoId/integrantes/:integranteId/rol", h.CambiarRolIntegrante)
	e.router.DELETE("/proyectos/:proyectoId/integrantes/:integranteId", h.QuitarIntegrante)

	ei := escenarioIntegrantes{escenarioVersiones: e}
	for usuario, destino := range map[int64]*int64{
		e.productor: &ei.integranteProductor,
		e.musico:    &ei.integranteMusico,
	} {
		if err := e.db.QueryRow(
			`SELECT codintegrante FROM integrante WHERE codigousuario = $1`, usuario,
		).Scan(destino); err != nil {
			t.Fatal(err)
		}
	}
	return ei
}

func (e escenarioIntegrantes) rutaIntegrantes(integrante int64) string {
	r := fmt.Sprintf("/proyectos/%d/integrantes", e.proyecto)
	if integrante != 0 {
		r += fmt.Sprintf("/%d", integrante)
	}
	return r
}

func (e escenarioIntegrantes) listarIntegrantes(t *testing.T, usuario int64) (int, []dto.ColaboradorProyectoResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, e.rutaIntegrantes(0), nil)
	req.Header.Set("X-Usuario", fmt.Sprint(usuario))
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	var integrantes []dto.ColaboradorProyectoResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &integrantes)
	return rec.Code, integrantes
}

func TestIntegrantes_ListarConRolYPropietario(t *testing.T) {
	e := prepararEscenarioIntegrantes(t)

	// Cualquier integrante los ve, no solo el propietario.
	code, integrantes := e.listarIntegrantes(t, e.musico)
	if code != http.StatusOK || len(integrantes) != 2 {
		t.Fatalf("status = %d, integrantes = %+v", code, integrantes)
	}
	for _, i := range integrantes {
		switch i.CodigoIntegrante {
		case e.integranteProductor:
			if !i.EsPropietario || i.CodRol != rolProductor || i.NombreRol != "Productor" || i.Nombre != "Ver productor" {
				t.Fatalf("productor = %+v", i)
			}
		case e.integranteMusico:
			if i.EsPropietario || i.CodRol != rolMusico || i.NombreRol == "" {
				t.Fatalf("músico = %+v", i)
			}
		default:
			t.Fatalf("integrante inesperado: %+v", i)
		}
	}

	code, _ = e.listarIntegrantes(t, e.nuevoAjeno(t))
	if code != http.StatusForbidden {
		t.Fatalf("ajeno: status = %d, quería 403", code)
	}
}

func TestIntegrantes_CambiarRolNoTocaLaMarcaDePropietario(t *testing.T) {
	e := prepararEscenarioIntegrantes(t)

	code, body := e.pedir(t, e.productor, http.MethodPatch,
		e.rutaIntegrantes(e.integranteMusico)+"/rol", map[string]any{"codRol": rolProductor})
	esperarStatus(t, code, body, http.StatusOK, "")

	_, integrantes := e.listarIntegrantes(t, e.productor)
	for _, i := range integrantes {
		if i.CodigoIntegrante == e.integranteMusico && (i.CodRol != rolProductor || i.EsPropietario) {
			t.Fatalf("después del cambio: %+v", i)
		}
	}
}

func TestIntegrantes_QuitarDejaAfueraAlIntegrante(t *testing.T) {
	e := prepararEscenarioIntegrantes(t)

	code, body := e.pedir(t, e.productor, http.MethodDelete, e.rutaIntegrantes(e.integranteMusico), nil)
	esperarStatus(t, code, body, http.StatusNoContent, "")

	_, integrantes := e.listarIntegrantes(t, e.productor)
	if len(integrantes) != 1 || integrantes[0].CodigoIntegrante != e.integranteProductor {
		t.Fatalf("integrantes después de quitar = %+v", integrantes)
	}
	if code, _ := e.listarIntegrantes(t, e.musico); code != http.StatusForbidden {
		t.Fatalf("el quitado sigue viendo el proyecto: status = %d", code)
	}
}

func TestIntegrantes_Errores(t *testing.T) {
	e := prepararEscenarioIntegrantes(t)
	soloPropietario := "Solo el propietario puede gestionar los integrantes del proyecto"

	casos := []struct {
		nombre  string
		usuario int64
		metodo  string
		ruta    string
		cuerpo  any
		status  int
		mensaje string
	}{
		{"rol vacío", e.productor, http.MethodPatch, e.rutaIntegrantes(e.integranteMusico) + "/rol",
			map[string]any{}, http.StatusBadRequest, "El rol es obligatorio"},
		{"rol de sistema", e.productor, http.MethodPatch, e.rutaIntegrantes(e.integranteMusico) + "/rol",
			map[string]any{"codRol": rolAdministradorSistema}, http.StatusBadRequest,
			"El rol seleccionado no es válido para un proyecto"},
		{"quitar al propietario", e.productor, http.MethodDelete, e.rutaIntegrantes(e.integranteProductor),
			nil, http.StatusConflict, "No se puede quitar al propietario del proyecto"},
		{"integrante de otro proyecto", e.productor, http.MethodDelete, e.rutaIntegrantes(999999999),
			nil, http.StatusNotFound, "El integrante no pertenece activamente al proyecto"},
		{"un músico cambia un rol", e.musico, http.MethodPatch, e.rutaIntegrantes(e.integranteProductor) + "/rol",
			map[string]any{"codRol": rolMusico}, http.StatusForbidden, soloPropietario},
		{"un músico quita a alguien", e.musico, http.MethodDelete, e.rutaIntegrantes(e.integranteProductor),
			nil, http.StatusForbidden, soloPropietario},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			code, body := e.pedir(t, caso.usuario, caso.metodo, caso.ruta, caso.cuerpo)
			esperarStatus(t, code, body, caso.status, caso.mensaje)
		})
	}
}
