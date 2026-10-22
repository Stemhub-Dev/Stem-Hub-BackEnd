package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// Tests de integración de HU-CFG-B06 (ABM de tipos de proyecto) y HU-CFG-B07
// (ABM de estados de proyecto): mismas rutas que en router.go, con el
// permiso real de la base (el rol Administrador tiene los dos).

type catalogoProyecto struct {
	nombre  string
	ruta    string
	permiso string
	tabla   string
	columna string
}

var catalogosProyecto = []catalogoProyecto{
	{"tipos", "/configuracion/tipos-proyecto", "GESTIONAR_TIPOS_PROYECTO", "tipoproyecto", "nombretipoproy"},
	{"estados", "/configuracion/estados-proyecto", "GESTIONAR_ESTADOS_PROYECTO", "estadoproyecto", "nombreestadoproy"},
}

type escenarioCatalogos struct {
	router *gin.Engine
	admin  int64
	comun  int64
	// Termina cada nombre creado, para borrarlos al limpiar.
	sufijo string
}

func prepararEscenarioCatalogos(t *testing.T) escenarioCatalogos {
	t.Helper()
	db := conectarBaseDePrueba(t)
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())

	nuevoUsuario := func(nombre string) int64 {
		var id int64
		if err := db.QueryRow(
			`INSERT INTO usuario (email, idautenticacion) VALUES ($1, gen_random_uuid()) RETURNING codigousuario`,
			"cfg-"+nombre+"-"+sufijo+"@test.local",
		).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	e := escenarioCatalogos{admin: nuevoUsuario("admin"), comun: nuevoUsuario("comun"), sufijo: sufijo}
	if _, err := db.Exec(
		`INSERT INTO usuariorol (codigousuario, codrol) VALUES ($1, $2)`, e.admin, rolAdministradorSistema,
	); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, c := range catalogosProyecto {
			_, _ = db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE %s LIKE $1`, c.tabla, c.columna), "%"+sufijo)
		}
		_, _ = db.Exec(`DELETE FROM usuariorol WHERE codigousuario = $1`, e.admin)
		_, _ = db.Exec(`DELETE FROM usuario WHERE codigousuario IN ($1, $2)`, e.admin, e.comun)
	})

	permisos := middleware.NewPermisoMiddleware(service.NewPermisoService(repository.NewPermisoRepository(db)))
	tipos := handler.NewTipoProyectoHandler(service.NewTipoProyectoService(repository.NewTipoProyectoRepository(db)))
	estados := handler.NewEstadoProyectoHandler(service.NewEstadoProyectoService(repository.NewEstadoProyectoRepository(db)))

	gin.SetMode(gin.TestMode)
	e.router = gin.New()
	e.router.Use(func(c *gin.Context) {
		var u model.Usuario
		fmt.Sscan(c.GetHeader("X-Usuario"), &u.CodigoUsuario)
		c.Set(middleware.UsuarioContextKey, &u)
	})
	registrar := func(ruta, permiso string, listar, crear, editar, cambiar gin.HandlerFunc) {
		e.router.GET(ruta, listar)
		e.router.POST(ruta, permisos.RequerirPermiso(permiso), crear)
		e.router.PUT(ruta+"/:id", permisos.RequerirPermiso(permiso), editar)
		e.router.PATCH(ruta+"/:id", permisos.RequerirPermiso(permiso), cambiar)
	}
	registrar(catalogosProyecto[0].ruta, catalogosProyecto[0].permiso, tipos.Listar, tipos.Crear, tipos.Editar, tipos.CambiarEstado)
	registrar(catalogosProyecto[1].ruta, catalogosProyecto[1].permiso, estados.Listar, estados.Crear, estados.Editar, estados.CambiarEstado)
	return e
}

type valorCatalogo struct {
	ID     int64  `json:"id"`
	Nombre string `json:"nombre"`
	Activo bool   `json:"activo"`
}

func (e escenarioCatalogos) pedir(t *testing.T, usuario int64, metodo, ruta string, cuerpo any) (int, []byte) {
	t.Helper()
	datos, _ := json.Marshal(cuerpo)
	req := httptest.NewRequest(metodo, ruta, bytes.NewReader(datos))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Usuario", fmt.Sprint(usuario))
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

func (e escenarioCatalogos) listar(t *testing.T, ruta string) []valorCatalogo {
	t.Helper()
	code, body := e.pedir(t, e.comun, http.MethodGet, ruta, nil)
	if code != http.StatusOK {
		t.Fatalf("listar %s: %d %s", ruta, code, body)
	}
	var valores []valorCatalogo
	if err := json.Unmarshal(body, &valores); err != nil {
		t.Fatal(err)
	}
	return valores
}

func buscar(valores []valorCatalogo, id int64) (valorCatalogo, bool) {
	for _, v := range valores {
		if v.ID == id {
			return v, true
		}
	}
	return valorCatalogo{}, false
}

func contiene(valores []valorCatalogo, id int64) bool {
	_, ok := buscar(valores, id)
	return ok
}

func TestCatalogosProyecto_AltaEdicionYEstado(t *testing.T) {
	e := prepararEscenarioCatalogos(t)

	for _, c := range catalogosProyecto {
		t.Run(c.nombre, func(t *testing.T) {
			nombre := fmt.Sprintf("Prueba %s %s", c.nombre, e.sufijo)

			code, body := e.pedir(t, e.admin, http.MethodPost, c.ruta, map[string]any{"nombre": nombre})
			if code != http.StatusCreated {
				t.Fatalf("crear: %d %s", code, body)
			}
			var creado valorCatalogo
			_ = json.Unmarshal(body, &creado)
			if creado.ID == 0 || creado.Nombre != nombre || !creado.Activo {
				t.Fatalf("creado = %+v", creado)
			}

			renombrado := "Renombrado " + nombre
			code, body = e.pedir(t, e.admin, http.MethodPut, fmt.Sprintf("%s/%d", c.ruta, creado.ID),
				map[string]any{"nombre": renombrado})
			if code != http.StatusOK {
				t.Fatalf("editar: %d %s", code, body)
			}
			if guardado, ok := buscar(e.listar(t, c.ruta), creado.ID); !ok || guardado.Nombre != renombrado {
				t.Fatalf("después de editar: %+v (encontrado = %v), quería el nombre %q", guardado, ok, renombrado)
			}

			code, body = e.pedir(t, e.admin, http.MethodPatch, fmt.Sprintf("%s/%d", c.ruta, creado.ID),
				map[string]any{"activo": false})
			if code != http.StatusOK {
				t.Fatalf("desactivar: %d %s", code, body)
			}

			// El listado común muestra solo los activos; con incluirInactivos, todos.
			if contiene(e.listar(t, c.ruta), creado.ID) {
				t.Fatal("el desactivado sigue en el listado de activos")
			}
			if !contiene(e.listar(t, c.ruta+"?incluirInactivos=true"), creado.ID) {
				t.Fatal("el desactivado no aparece con incluirInactivos=true")
			}

			code, body = e.pedir(t, e.admin, http.MethodPatch, fmt.Sprintf("%s/%d", c.ruta, creado.ID),
				map[string]any{"activo": true})
			if code != http.StatusOK || !contiene(e.listar(t, c.ruta), creado.ID) {
				t.Fatalf("reactivar: %d %s", code, body)
			}
		})
	}
}

func TestCatalogosProyecto_Errores(t *testing.T) {
	e := prepararEscenarioCatalogos(t)

	for _, c := range catalogosProyecto {
		t.Run(c.nombre, func(t *testing.T) {
			casos := []struct {
				nombre  string
				usuario int64
				metodo  string
				ruta    string
				cuerpo  any
				status  int
			}{
				{"nombre vacío", e.admin, http.MethodPost, c.ruta, map[string]any{"nombre": "  "}, http.StatusBadRequest},
				{"id inválido al editar", e.admin, http.MethodPut, c.ruta + "/abc", map[string]any{"nombre": "X"}, http.StatusBadRequest},
				{"id inválido al cambiar el estado", e.admin, http.MethodPatch, c.ruta + "/abc", map[string]any{"activo": false}, http.StatusBadRequest},
				{"sin permiso crea", e.comun, http.MethodPost, c.ruta, map[string]any{"nombre": "X"}, http.StatusForbidden},
				{"sin permiso edita", e.comun, http.MethodPut, c.ruta + "/1", map[string]any{"nombre": "X"}, http.StatusForbidden},
				{"sin permiso cambia el estado", e.comun, http.MethodPatch, c.ruta + "/1", map[string]any{"activo": false}, http.StatusForbidden},
			}
			for _, caso := range casos {
				t.Run(caso.nombre, func(t *testing.T) {
					code, body := e.pedir(t, caso.usuario, caso.metodo, caso.ruta, caso.cuerpo)
					if code != caso.status {
						t.Fatalf("status = %d, quería %d (%s)", code, caso.status, body)
					}
				})
			}
		})
	}
}
