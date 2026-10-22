package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// Tests de integración de editar y dar de baja una versión (HU-ABM-03-02 y
// HU-ABM-03-03) contra PostgreSQL real (mismo TEST_DATABASE_URL que
// mis_canciones_integracion_test.go).

type escenarioVersiones struct {
	db        *sql.DB
	router    *gin.Engine
	proyecto  int64
	cancion   int64
	productor int64
	musico    int64
	versiones []int64 // v1, v2, v3 (la actual)
}

func prepararEscenarioVersiones(t *testing.T) *escenarioVersiones {
	t.Helper()

	db := conectarBaseDePrueba(t)
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())
	e := &escenarioVersiones{db: db}
	var usuarios, integrantes []int64

	t.Cleanup(func() {
		ejecutar := func(consulta string, args ...any) {
			if _, err := db.Exec(consulta, args...); err != nil {
				t.Errorf("limpieza (%s): %v", consulta, err)
			}
		}
		ejecutar(`DELETE FROM cancionversion WHERE codigocancion = $1`, e.cancion)
		ejecutar(`DELETE FROM cancion WHERE codigocancion = $1`, e.cancion)
		ejecutar(`DELETE FROM integranteproyecto WHERE codigoproyecto = $1`, e.proyecto)
		ejecutar(`DELETE FROM proyecto WHERE codigoproyecto = $1`, e.proyecto)
		for _, id := range integrantes {
			ejecutar(`DELETE FROM integrante WHERE codintegrante = $1`, id)
		}
		for _, id := range usuarios {
			ejecutar(`DELETE FROM usuario WHERE codigousuario = $1`, id)
		}
	})

	debe := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("carga del escenario: %v", err)
		}
	}

	nuevo := func(nombre string) (int64, int64) {
		var usuario, integrante int64
		debe(db.QueryRow(
			`INSERT INTO usuario (email, idautenticacion) VALUES ($1, gen_random_uuid()) RETURNING codigousuario`,
			"ver-"+nombre+"-"+sufijo+"@test.local",
		).Scan(&usuario))
		usuarios = append(usuarios, usuario)
		debe(db.QueryRow(
			`INSERT INTO integrante (codigousuario, nombreintegrante) VALUES ($1, $2) RETURNING codintegrante`,
			usuario, "Ver "+nombre,
		).Scan(&integrante))
		integrantes = append(integrantes, integrante)
		return usuario, integrante
	}

	var integranteProductor, integranteMusico int64
	e.productor, integranteProductor = nuevo("productor")
	e.musico, integranteMusico = nuevo("musico")

	debe(db.QueryRow(
		`INSERT INTO proyecto (nombreproyecto, codtipoproy, codestadoproy) VALUES ($1, $2, $3) RETURNING codigoproyecto`,
		"Proyecto versiones "+sufijo, tipoSingle, estadoSinEmpezar,
	).Scan(&e.proyecto))
	_, err := db.Exec(
		`INSERT INTO integranteproyecto (codintegrante, codigoproyecto, codrol, espropietario)
		 VALUES ($1, $3, $4, TRUE), ($2, $3, $5, FALSE)`,
		integranteProductor, integranteMusico, e.proyecto, rolProductor, rolMusico,
	)
	debe(err)

	debe(db.QueryRow(
		`INSERT INTO cancion (codigoproyecto, nombrecancion) VALUES ($1, 'Tema') RETURNING codigocancion`,
		e.proyecto,
	).Scan(&e.cancion))

	for numero := 1; numero <= 3; numero++ {
		var id int64
		debe(db.QueryRow(`
			INSERT INTO cancionversion
			  (codigocancion, numeroversion, fechahoraaltaversion, urlarchivocancionver,
			   formatoarchivocancionver, notasversion, codintegrantealtaversion)
			VALUES ($1, $2, $3, $4, 'mp3', $5, $6) RETURNING codigocancionversion`,
			e.cancion, numero, time.Now().Add(time.Duration(numero-4)*time.Hour),
			fmt.Sprintf("proyectos/%d/v%d.mp3", e.proyecto, numero),
			fmt.Sprintf("notas v%d", numero), integranteProductor,
		).Scan(&id))
		e.versiones = append(e.versiones, id)
	}

	svc := service.NewCancionService(
		repository.NewCancionRepository(db),
		repository.NewProyectoRepository(db),
		repository.NewIntegranteRepository(db),
		storageDePrueba{},
	)
	h := handler.NewCancionHandler(svc)

	gin.SetMode(gin.TestMode)
	e.router = gin.New()
	e.router.Use(func(c *gin.Context) {
		var u model.Usuario
		fmt.Sscan(c.GetHeader("X-Usuario"), &u.CodigoUsuario)
		c.Set(middleware.UsuarioContextKey, &u)
	})
	base := "/proyectos/:proyectoId/canciones/:cancionId/versiones"
	e.router.GET(base, h.ListarVersiones)
	e.router.POST(base, h.CrearVersion)
	e.router.PATCH(base+"/:versionId", h.EditarVersion)
	e.router.DELETE(base+"/:versionId", h.DarDeBajaVersion)
	e.router.GET(base+"/:versionId/audio", h.ObtenerAudioVersion)

	return e
}

func (e *escenarioVersiones) ruta(version int64) string {
	r := fmt.Sprintf("/proyectos/%d/canciones/%d/versiones", e.proyecto, e.cancion)
	if version != 0 {
		r += fmt.Sprintf("/%d", version)
	}
	return r
}

func (e *escenarioVersiones) pedir(t *testing.T, usuario int64, metodo, ruta string, cuerpo any) (int, map[string]any) {
	t.Helper()
	var lector *bytes.Reader
	if cuerpo != nil {
		datos, _ := json.Marshal(cuerpo)
		lector = bytes.NewReader(datos)
	} else {
		lector = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(metodo, ruta, lector)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Usuario", fmt.Sprint(usuario))
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

type versionListada struct {
	CodigoCancionVersion int64   `json:"codigoCancionVersion"`
	NumeroVersion        int     `json:"numeroVersion"`
	Notas                *string `json:"notas"`
	EsActual             bool    `json:"esActual"`
	FechaHoraAlta        string  `json:"fechaHoraAlta"`
	URLArchivo           *string `json:"urlArchivo"`
	Autor                *struct {
		Nombre string `json:"nombre"`
	} `json:"autor"`
}

func (e *escenarioVersiones) listar(t *testing.T) []versionListada {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, e.ruta(0), nil)
	req.Header.Set("X-Usuario", fmt.Sprint(e.productor))
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("listar: %d %s", rec.Code, rec.Body.String())
	}
	var versiones []versionListada
	if err := json.Unmarshal(rec.Body.Bytes(), &versiones); err != nil {
		t.Fatal(err)
	}
	return versiones
}

func esperarStatus(t *testing.T, code int, body map[string]any, quiero int, mensaje string) {
	t.Helper()
	if code != quiero {
		t.Fatalf("status = %d, quería %d (%v)", code, quiero, body)
	}
	if mensaje != "" && body["error"] != mensaje {
		t.Fatalf("error = %q, quería %q", body["error"], mensaje)
	}
}

// HU-ABM-03-02

func TestVersiones_EditarNotas(t *testing.T) {
	e := prepararEscenarioVersiones(t)
	v2 := e.versiones[1]
	antes := e.listar(t)

	code, body := e.pedir(t, e.productor, http.MethodPatch, e.ruta(v2), map[string]any{"notas": "  Mezcla nueva  "})
	esperarStatus(t, code, body, http.StatusOK, "")
	if body["notas"] != "Mezcla nueva" {
		t.Fatalf("respuesta = %v", body)
	}

	despues := e.listar(t)
	for i := range despues {
		a, d := antes[i], despues[i]
		if d.CodigoCancionVersion == v2 {
			if d.Notas == nil || *d.Notas != "Mezcla nueva" {
				t.Fatalf("notas de v2 = %v", d.Notas)
			}
			// Número, autor, fecha y pista no cambian.
			if d.NumeroVersion != a.NumeroVersion || d.FechaHoraAlta != a.FechaHoraAlta ||
				*d.URLArchivo != *a.URLArchivo || d.Autor.Nombre != a.Autor.Nombre || d.EsActual != a.EsActual {
				t.Fatalf("cambió algo más que las notas: antes %+v, después %+v", a, d)
			}
		} else if *d.Notas != *a.Notas {
			t.Fatalf("cambiaron las notas de otra versión: %+v", d)
		}
	}

	// Vacías borran las notas.
	code, body = e.pedir(t, e.productor, http.MethodPatch, e.ruta(v2), map[string]any{"notas": "   "})
	esperarStatus(t, code, body, http.StatusOK, "")
	if body["notas"] != nil {
		t.Fatalf("notas vacías = %v", body["notas"])
	}
}

func TestVersiones_EditarErrores(t *testing.T) {
	e := prepararEscenarioVersiones(t)
	v2 := e.versiones[1]

	code, body := e.pedir(t, e.musico, http.MethodPatch, e.ruta(v2), map[string]any{"notas": "x"})
	esperarStatus(t, code, body, http.StatusForbidden, "No tenés permiso para modificar versiones")

	code, body = e.pedir(t, e.productor, http.MethodPatch, e.ruta(v2), map[string]any{})
	esperarStatus(t, code, body, http.StatusBadRequest, "Solicitud inválida")

	code, body = e.pedir(t, e.productor, http.MethodPatch, e.ruta(999999999), map[string]any{"notas": "x"})
	esperarStatus(t, code, body, http.StatusNotFound, "La versión no existe")

	code, body = e.pedir(t, e.productor, http.MethodPatch,
		fmt.Sprintf("/proyectos/%d/canciones/999999999/versiones/%d", e.proyecto, v2), map[string]any{"notas": "x"})
	esperarStatus(t, code, body, http.StatusNotFound, "La canción no existe en el proyecto")

	code, body = e.pedir(t, e.productor, http.MethodPatch,
		fmt.Sprintf("/proyectos/999999999/canciones/%d/versiones/%d", e.cancion, v2), map[string]any{"notas": "x"})
	esperarStatus(t, code, body, http.StatusNotFound, "El proyecto no existe")

	code, body = e.pedir(t, e.productor, http.MethodPatch,
		fmt.Sprintf("/proyectos/%d/canciones/%d/versiones/abc", e.proyecto, e.cancion), map[string]any{"notas": "x"})
	esperarStatus(t, code, body, http.StatusBadRequest, "Versión inválida")
}

// HU-ABM-03-03

func TestVersiones_BajaDeLaActual(t *testing.T) {
	e := prepararEscenarioVersiones(t)
	v1, v2, v3 := e.versiones[0], e.versiones[1], e.versiones[2]

	code, body := e.pedir(t, e.musico, http.MethodDelete, e.ruta(v3), nil)
	esperarStatus(t, code, body, http.StatusForbidden, "No tenés permiso para dar de baja versiones en este proyecto")

	code, body = e.pedir(t, e.productor, http.MethodDelete, e.ruta(v3), nil)
	esperarStatus(t, code, body, http.StatusOK, "")
	if body["numeroVersion"] != float64(3) {
		t.Fatalf("respuesta = %v", body)
	}

	// Deja de listarse y la anterior más reciente pasa a ser la actual.
	versiones := e.listar(t)
	if len(versiones) != 2 || versiones[0].CodigoCancionVersion != v2 || !versiones[0].EsActual || versiones[1].EsActual {
		t.Fatalf("versiones después de la baja = %+v", versiones)
	}

	// Baja lógica: la fila sigue.
	var baja sql.NullTime
	if err := e.db.QueryRow(`SELECT fechahorabajaversion FROM cancionversion WHERE codigocancionversion = $1`, v3).Scan(&baja); err != nil || !baja.Valid {
		t.Fatalf("baja lógica: %v %v", baja, err)
	}

	// Ya dada de baja: 404, también para editarla.
	code, body = e.pedir(t, e.productor, http.MethodDelete, e.ruta(v3), nil)
	esperarStatus(t, code, body, http.StatusNotFound, "La versión no existe")
	code, body = e.pedir(t, e.productor, http.MethodPatch, e.ruta(v3), map[string]any{"notas": "x"})
	esperarStatus(t, code, body, http.StatusNotFound, "La versión no existe")

	// Queda v1 sola tras borrar v2: es la única, 409.
	code, body = e.pedir(t, e.productor, http.MethodDelete, e.ruta(v2), nil)
	esperarStatus(t, code, body, http.StatusOK, "")
	code, body = e.pedir(t, e.productor, http.MethodDelete, e.ruta(v1), nil)
	esperarStatus(t, code, body, http.StatusConflict, "Una canción necesita al menos una versión")
	if versiones := e.listar(t); len(versiones) != 1 || !versiones[0].EsActual {
		t.Fatalf("queda = %+v", versiones)
	}
}

// Tras borrar la última versión, la siguiente no reutiliza su número (el
// índice único incluye las dadas de baja y la clave del archivo usa el número).
func TestVersiones_NuevaVersionDespuesDeBorrarLaUltima(t *testing.T) {
	e := prepararEscenarioVersiones(t)

	code, body := e.pedir(t, e.productor, http.MethodDelete, e.ruta(e.versiones[2]), nil)
	esperarStatus(t, code, body, http.StatusOK, "")

	var cuerpo bytes.Buffer
	formulario := multipart.NewWriter(&cuerpo)
	archivo, _ := formulario.CreateFormFile("archivo", "pista.mp3")
	_, _ = archivo.Write([]byte("audio de prueba"))
	_ = formulario.WriteField("notas", "re-subida")
	_ = formulario.Close()

	req := httptest.NewRequest(http.MethodPost, e.ruta(0), &cuerpo)
	req.Header.Set("Content-Type", formulario.FormDataContentType())
	req.Header.Set("X-Usuario", fmt.Sprint(e.productor))
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("crear versión: %d %s", rec.Code, rec.Body.String())
	}
	var creada map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &creada)
	if creada["numeroVersion"] != float64(4) {
		t.Fatalf("número de la nueva versión = %v, quería 4", creada["numeroVersion"])
	}
	if versiones := e.listar(t); versiones[0].NumeroVersion != 4 || !versiones[0].EsActual {
		t.Fatalf("versiones = %+v", versiones)
	}
}

// Dos bajas en paralelo de las dos únicas versiones: una sola prospera.
func TestVersiones_BajasEnParaleloNoDejanSinVersiones(t *testing.T) {
	e := prepararEscenarioVersiones(t)

	code, body := e.pedir(t, e.productor, http.MethodDelete, e.ruta(e.versiones[0]), nil)
	esperarStatus(t, code, body, http.StatusOK, "")

	var wg sync.WaitGroup
	largada := make(chan struct{})
	codigos := make([]int, 2)
	for i, v := range e.versiones[1:] {
		wg.Add(1)
		go func(i int, v int64) {
			defer wg.Done()
			<-largada
			codigos[i], _ = e.pedir(t, e.productor, http.MethodDelete, e.ruta(v), nil)
		}(i, v)
	}
	close(largada)
	wg.Wait()

	if !(codigos[0] == http.StatusOK && codigos[1] == http.StatusConflict) &&
		!(codigos[0] == http.StatusConflict && codigos[1] == http.StatusOK) {
		t.Fatalf("códigos = %v, quería un 200 y un 409", codigos)
	}
	if versiones := e.listar(t); len(versiones) != 1 {
		t.Fatalf("quedaron %d versiones", len(versiones))
	}
}
