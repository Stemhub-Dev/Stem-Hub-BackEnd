package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/mailer"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// Tests de integración de invitaciones (HU-NOT-B01, HU-NOT-B02 y el enlace
// copiable de HU-NOT-F02) contra PostgreSQL real, con el mismo
// TEST_DATABASE_URL que mis_canciones_integracion_test.go. El mailer es
// falso: guarda los mails o falla a pedido.

const rolAdministradorSistema int64 = 3

type mailerDePrueba struct {
	fallar   bool
	enviados []mailer.InvitacionEmailData
}

func (m *mailerDePrueba) EnviarInvitacionProyecto(data mailer.InvitacionEmailData) error {
	if m.fallar {
		return errors.New("smtp caído")
	}
	m.enviados = append(m.enviados, data)
	return nil
}

type usuarioInvitacion struct {
	codigo int64
	email  string
}

type escenarioInvitaciones struct {
	db       *sql.DB
	mailer   *mailerDePrueba
	router   *gin.Engine
	proyecto int64

	propietario usuarioInvitacion
	integrante  usuarioInvitacion // ya es Músico del proyecto
	invitado    usuarioInvitacion // tiene cuenta, no está en el proyecto
	otro        usuarioInvitacion // tiene cuenta, no está en el proyecto
}

func prepararEscenarioInvitaciones(t *testing.T) *escenarioInvitaciones {
	t.Helper()

	db := conectarBaseDePrueba(t)
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())

	var usuarios, integrantes []int64
	var proyecto int64

	t.Cleanup(func() {
		ejecutar := func(consulta string, args ...any) {
			if _, err := db.Exec(consulta, args...); err != nil {
				t.Errorf("limpieza (%s): %v", consulta, err)
			}
		}
		ejecutar(`DELETE FROM invitacionproyecto WHERE codigoproyecto = $1`, proyecto)
		ejecutar(`DELETE FROM integranteproyecto WHERE codigoproyecto = $1`, proyecto)
		ejecutar(`DELETE FROM proyecto WHERE codigoproyecto = $1`, proyecto)
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

	nuevoUsuario := func(nombre string) (usuarioInvitacion, int64) {
		u := usuarioInvitacion{email: "inv-" + nombre + "-" + sufijo + "@test.local"}
		debe(db.QueryRow(
			`INSERT INTO usuario (email, idautenticacion)
			 VALUES ($1, gen_random_uuid()) RETURNING codigousuario`,
			u.email,
		).Scan(&u.codigo))
		usuarios = append(usuarios, u.codigo)

		var integrante int64
		debe(db.QueryRow(
			`INSERT INTO integrante (codigousuario, nombreintegrante)
			 VALUES ($1, $2) RETURNING codintegrante`,
			u.codigo, "Inv "+nombre,
		).Scan(&integrante))
		integrantes = append(integrantes, integrante)
		return u, integrante
	}

	propietario, integrantePropietario := nuevoUsuario("propietario")
	integrante, integranteMusico := nuevoUsuario("integrante")
	invitado, _ := nuevoUsuario("invitado")
	otro, _ := nuevoUsuario("otro")

	debe(db.QueryRow(
		`INSERT INTO proyecto (nombreproyecto, codtipoproy, codestadoproy)
		 VALUES ($1, $2, $3) RETURNING codigoproyecto`,
		"Proyecto invitaciones "+sufijo, tipoSingle, estadoSinEmpezar,
	).Scan(&proyecto))

	_, err := db.Exec(
		`INSERT INTO integranteproyecto (codintegrante, codigoproyecto, codrol, espropietario)
		 VALUES ($1, $3, $4, TRUE), ($2, $3, $5, FALSE)`,
		integrantePropietario, integranteMusico, proyecto, rolProductor, rolMusico,
	)
	debe(err)

	m := &mailerDePrueba{}
	svc := service.NewInvitacionProyectoService(
		repository.NewInvitacionProyectoRepository(db),
		repository.NewProyectoRepository(db),
		repository.NewIntegranteRepository(db),
		repository.NewUsuarioRepository(db),
		repository.NewRolRepository(db),
		m,
		"http://front.test",
		7,
	)
	h := handler.NewInvitacionProyectoHandler(svc)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	// El usuario de la request sale de los headers de prueba (en la app lo
	// pone el middleware JWT).
	router.Use(func(c *gin.Context) {
		if codigo := c.GetHeader("X-Usuario"); codigo != "" {
			var u model.Usuario
			fmt.Sscan(codigo, &u.CodigoUsuario)
			u.Email = c.GetHeader("X-Email")
			c.Set(middleware.UsuarioContextKey, &u)
		}
	})
	router.POST("/proyectos/:proyectoId/invitaciones", h.Crear)
	router.POST("/proyectos/:proyectoId/invitaciones/enlace", h.CrearEnlace)
	router.GET("/proyectos/:proyectoId/invitaciones", h.ListarPendientes)
	router.DELETE("/proyectos/:proyectoId/invitaciones/:invitacionId", h.Cancelar)
	router.GET("/invitaciones/:token", h.ObtenerDetalle)
	router.GET("/invitaciones", h.MisInvitaciones)
	router.POST("/invitaciones/:token/aceptar", h.Aceptar)
	router.POST("/invitaciones/:token/rechazar", h.Rechazar)

	return &escenarioInvitaciones{
		db:          db,
		mailer:      m,
		router:      router,
		proyecto:    proyecto,
		propietario: propietario,
		integrante:  integrante,
		invitado:    invitado,
		otro:        otro,
	}
}

type respuestaInvitacion struct {
	code int
	body map[string]any
	list []map[string]any
}

func (e *escenarioInvitaciones) pedir(
	t *testing.T,
	como *usuarioInvitacion,
	metodo, ruta string,
	cuerpo any,
) respuestaInvitacion {
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
	if como != nil {
		req.Header.Set("X-Usuario", fmt.Sprint(como.codigo))
		req.Header.Set("X-Email", como.email)
	}

	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)

	r := respuestaInvitacion{code: rec.Code}
	if strings.HasPrefix(strings.TrimSpace(rec.Body.String()), "[") {
		_ = json.Unmarshal(rec.Body.Bytes(), &r.list)
	} else {
		_ = json.Unmarshal(rec.Body.Bytes(), &r.body)
	}
	return r
}

func (e *escenarioInvitaciones) rutaInvitaciones() string {
	return fmt.Sprintf("/proyectos/%d/invitaciones", e.proyecto)
}

func esperar(t *testing.T, r respuestaInvitacion, code int, errorEsperado string) {
	t.Helper()
	if r.code != code {
		t.Fatalf("status = %d, quería %d (body %v)", r.code, code, r.body)
	}
	if errorEsperado != "" && r.body["error"] != errorEsperado {
		t.Fatalf("error = %q, quería %q", r.body["error"], errorEsperado)
	}
}

func tokenDe(t *testing.T, r respuestaInvitacion) string {
	t.Helper()
	enlace, _ := r.body["enlace"].(string)
	const prefijo = "http://front.test/invitaciones/"
	if !strings.HasPrefix(enlace, prefijo) || len(enlace) == len(prefijo) {
		t.Fatalf("enlace = %q, quería %s<token>", enlace, prefijo)
	}
	return strings.TrimPrefix(enlace, prefijo)
}

// HU-NOT-B01

func TestInvitaciones_CrearPorEmail(t *testing.T) {
	e := prepararEscenarioInvitaciones(t)

	r := e.pedir(t, &e.propietario, http.MethodPost, e.rutaInvitaciones(),
		map[string]any{"email": "  Nuevo@Test.Local ", "codRol": rolMusico})

	esperar(t, r, http.StatusCreated, "")
	if r.body["email"] != "nuevo@test.local" || r.body["estado"] != "pendiente" ||
		r.body["nombreRol"] != model.NombreRolMusicoArtista || r.body["emailEnviado"] != true ||
		r.body["porEnlace"] != false {
		t.Fatalf("respuesta = %v", r.body)
	}

	expiracion, err := time.Parse(time.RFC3339Nano, r.body["fechaHoraExpiracion"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Until(expiracion); d < 7*24*time.Hour-time.Minute || d > 7*24*time.Hour+time.Minute {
		t.Fatalf("vence en %v, quería 7 días", d)
	}

	token := tokenDe(t, r)
	if len(e.mailer.enviados) != 1 ||
		e.mailer.enviados[0].EmailDestino != "nuevo@test.local" ||
		!strings.HasSuffix(e.mailer.enviados[0].LinkAceptacion, "/invitaciones/"+token) {
		t.Fatalf("mails = %+v", e.mailer.enviados)
	}

	// Reinvitar al mismo email reutiliza la invitación con un token nuevo.
	r2 := e.pedir(t, &e.propietario, http.MethodPost, e.rutaInvitaciones(),
		map[string]any{"email": "nuevo@test.local", "codRol": rolProductor})
	esperar(t, r2, http.StatusCreated, "")
	if r2.body["codigoInvitacionProy"] != r.body["codigoInvitacionProy"] || tokenDe(t, r2) == token {
		t.Fatalf("reinvitación = %v, primera = %v", r2.body, r.body)
	}
}

func TestInvitaciones_FallaElMail(t *testing.T) {
	e := prepararEscenarioInvitaciones(t)
	e.mailer.fallar = true

	r := e.pedir(t, &e.propietario, http.MethodPost, e.rutaInvitaciones(),
		map[string]any{"email": "sin-mail@test.local", "codRol": rolMusico})

	// La invitación queda creada y el front puede ofrecer el enlace.
	esperar(t, r, http.StatusCreated, "")
	if r.body["emailEnviado"] != false {
		t.Fatalf("emailEnviado = %v, quería false", r.body["emailEnviado"])
	}
	token := tokenDe(t, r)

	detalle := e.pedir(t, nil, http.MethodGet, "/invitaciones/"+token, nil)
	esperar(t, detalle, http.StatusOK, "")
}

func TestInvitaciones_Validaciones(t *testing.T) {
	e := prepararEscenarioInvitaciones(t)
	ruta := e.rutaInvitaciones()

	largo := strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 63)
	for _, email := range []string{"", "no-es-email", "a@b", "Ana <ana@test.local>", largo} {
		r := e.pedir(t, &e.propietario, http.MethodPost, ruta, map[string]any{"email": email, "codRol": rolMusico})
		esperar(t, r, http.StatusBadRequest, "El email es inválido")
	}

	r := e.pedir(t, &e.propietario, http.MethodPost, ruta,
		map[string]any{"email": "x@test.local", "codRol": rolAdministradorSistema})
	esperar(t, r, http.StatusBadRequest, "El rol seleccionado no es válido para un proyecto")

	r = e.pedir(t, &e.propietario, http.MethodPost, ruta,
		map[string]any{"email": strings.ToUpper(e.integrante.email), "codRol": rolMusico})
	esperar(t, r, http.StatusConflict, "Ese usuario ya es integrante del proyecto")

	if len(e.mailer.enviados) != 0 {
		t.Fatalf("no tendría que haber mandado mails: %+v", e.mailer.enviados)
	}
}

func TestInvitaciones_PermisosYProyecto(t *testing.T) {
	e := prepararEscenarioInvitaciones(t)
	ruta := e.rutaInvitaciones()
	cuerpo := map[string]any{"email": "x@test.local", "codRol": rolMusico}

	// Un integrante que no es propietario.
	esperar(t, e.pedir(t, &e.integrante, http.MethodPost, ruta, cuerpo),
		http.StatusForbidden, "Solo el propietario del proyecto puede invitar")
	esperar(t, e.pedir(t, &e.integrante, http.MethodPost, ruta+"/enlace", map[string]any{"codRol": rolMusico}),
		http.StatusForbidden, "Solo el propietario del proyecto puede invitar")
	esperar(t, e.pedir(t, &e.integrante, http.MethodGet, ruta, nil),
		http.StatusForbidden, "Solo el propietario del proyecto puede ver las invitaciones")
	esperar(t, e.pedir(t, &e.integrante, http.MethodDelete, ruta+"/1", nil),
		http.StatusForbidden, "Solo el propietario del proyecto puede cancelar invitaciones")

	inexistente := "/proyectos/999999999/invitaciones"
	esperar(t, e.pedir(t, &e.propietario, http.MethodPost, inexistente, cuerpo),
		http.StatusNotFound, "El proyecto no existe")
	esperar(t, e.pedir(t, &e.propietario, http.MethodGet, inexistente, nil),
		http.StatusNotFound, "El proyecto no existe")
	esperar(t, e.pedir(t, &e.propietario, http.MethodDelete, inexistente+"/1", nil),
		http.StatusNotFound, "El proyecto no existe")
}

func TestInvitaciones_ListarYCancelar(t *testing.T) {
	e := prepararEscenarioInvitaciones(t)
	ruta := e.rutaInvitaciones()

	creada := e.pedir(t, &e.propietario, http.MethodPost, ruta,
		map[string]any{"email": e.invitado.email, "codRol": rolMusico})
	esperar(t, creada, http.StatusCreated, "")
	enlace := e.pedir(t, &e.propietario, http.MethodPost, ruta+"/enlace", map[string]any{"codRol": rolProductor})
	esperar(t, enlace, http.StatusCreated, "")

	lista := e.pedir(t, &e.propietario, http.MethodGet, ruta, nil)
	esperar(t, lista, http.StatusOK, "")
	if len(lista.list) != 2 {
		t.Fatalf("pendientes = %v, quería 2", lista.list)
	}

	id := fmt.Sprint(creada.body["codigoInvitacionProy"])
	esperar(t, e.pedir(t, &e.propietario, http.MethodDelete, ruta+"/"+id, nil), http.StatusNoContent, "")
	esperar(t, e.pedir(t, &e.propietario, http.MethodDelete, ruta+"/"+id, nil),
		http.StatusNotFound, "La invitación no existe")

	lista = e.pedir(t, &e.propietario, http.MethodGet, ruta, nil)
	if len(lista.list) != 1 || lista.list[0]["porEnlace"] != true {
		t.Fatalf("pendientes después de cancelar = %v", lista.list)
	}
}

// HU-NOT-B02

func TestInvitaciones_DetalleAceptarYMisInvitaciones(t *testing.T) {
	e := prepararEscenarioInvitaciones(t)

	creada := e.pedir(t, &e.propietario, http.MethodPost, e.rutaInvitaciones(),
		map[string]any{"email": e.invitado.email, "codRol": rolMusico})
	esperar(t, creada, http.StatusCreated, "")
	token := tokenDe(t, creada)

	detalle := e.pedir(t, nil, http.MethodGet, "/invitaciones/"+token, nil)
	esperar(t, detalle, http.StatusOK, "")
	if detalle.body["email"] != e.invitado.email || detalle.body["invitadoPor"] != "Inv propietario" ||
		detalle.body["nombreRol"] != model.NombreRolMusicoArtista || detalle.body["vencida"] != false ||
		detalle.body["yaAceptada"] != false || detalle.body["cancelada"] != false {
		t.Fatalf("detalle = %v", detalle.body)
	}
	esperar(t, e.pedir(t, nil, http.MethodGet, "/invitaciones/no-existe", nil),
		http.StatusNotFound, "La invitación no existe")

	mias := e.pedir(t, &e.invitado, http.MethodGet, "/invitaciones", nil)
	esperar(t, mias, http.StatusOK, "")
	if len(mias.list) != 1 || mias.list[0]["token"] != token {
		t.Fatalf("mis invitaciones = %v", mias.list)
	}

	esperar(t, e.pedir(t, &e.otro, http.MethodPost, "/invitaciones/"+token+"/aceptar", nil),
		http.StatusForbidden, "El email de tu cuenta no coincide con el de la invitación")

	aceptada := e.pedir(t, &e.invitado, http.MethodPost, "/invitaciones/"+token+"/aceptar", nil)
	esperar(t, aceptada, http.StatusOK, "")
	if aceptada.body["codigoProyecto"] != float64(e.proyecto) {
		t.Fatalf("aceptar = %v", aceptada.body)
	}

	var rol int64
	if err := e.db.QueryRow(`
		SELECT ip.codrol FROM integranteproyecto ip
		JOIN integrante i ON i.codintegrante = ip.codintegrante
		WHERE i.codigousuario = $1 AND ip.codigoproyecto = $2 AND ip.fechahorabajaintegranteproy IS NULL
	`, e.invitado.codigo, e.proyecto).Scan(&rol); err != nil || rol != rolMusico {
		t.Fatalf("membresía: rol %d, err %v", rol, err)
	}

	esperar(t, e.pedir(t, &e.invitado, http.MethodPost, "/invitaciones/"+token+"/aceptar", nil),
		http.StatusConflict, "La invitación ya fue aceptada")
}

func TestInvitaciones_RechazarVencidaYCancelada(t *testing.T) {
	e := prepararEscenarioInvitaciones(t)
	ruta := e.rutaInvitaciones()

	rechazo := e.pedir(t, &e.propietario, http.MethodPost, ruta, map[string]any{"email": e.invitado.email, "codRol": rolMusico})
	token := tokenDe(t, rechazo)
	esperar(t, e.pedir(t, &e.invitado, http.MethodPost, "/invitaciones/"+token+"/rechazar", nil),
		http.StatusNoContent, "")
	esperar(t, e.pedir(t, &e.invitado, http.MethodPost, "/invitaciones/"+token+"/aceptar", nil),
		http.StatusGone, "La invitación fue cancelada")

	vencida := e.pedir(t, &e.propietario, http.MethodPost, ruta, map[string]any{"email": e.otro.email, "codRol": rolMusico})
	tokenVencido := tokenDe(t, vencida)
	if _, err := e.db.Exec(
		`UPDATE invitacionproyecto SET fechahoraexpiracioninvitacion = CURRENT_TIMESTAMP - INTERVAL '1 hour'
		 WHERE tokeninvitacion = $1`, tokenVencido,
	); err != nil {
		t.Fatal(err)
	}
	esperar(t, e.pedir(t, &e.otro, http.MethodPost, "/invitaciones/"+tokenVencido+"/aceptar", nil),
		http.StatusGone, "La invitación venció")
	detalle := e.pedir(t, nil, http.MethodGet, "/invitaciones/"+tokenVencido, nil)
	if detalle.body["vencida"] != true {
		t.Fatalf("detalle vencida = %v", detalle.body)
	}
}

// HU-NOT-F02 "Copiar enlace"

func TestInvitaciones_Enlace(t *testing.T) {
	e := prepararEscenarioInvitaciones(t)
	ruta := e.rutaInvitaciones() + "/enlace"

	esperar(t, e.pedir(t, &e.propietario, http.MethodPost, ruta, map[string]any{"codRol": rolAdministradorSistema}),
		http.StatusBadRequest, "El rol seleccionado no es válido para un proyecto")
	esperar(t, e.pedir(t, &e.propietario, http.MethodPost, ruta, map[string]any{}),
		http.StatusBadRequest, "Solicitud inválida")

	creada := e.pedir(t, &e.propietario, http.MethodPost, ruta, map[string]any{"codRol": rolProductor})
	esperar(t, creada, http.StatusCreated, "")
	if creada.body["porEnlace"] != true || creada.body["email"] != "" ||
		creada.body["nombreRol"] != model.NombreRolProductor {
		t.Fatalf("enlace = %v", creada.body)
	}
	if _, enviado := creada.body["emailEnviado"]; enviado || len(e.mailer.enviados) != 0 {
		t.Fatalf("el enlace no manda mail: %v / %+v", creada.body, e.mailer.enviados)
	}
	token := tokenDe(t, creada)

	detalle := e.pedir(t, nil, http.MethodGet, "/invitaciones/"+token, nil)
	if detalle.body["porEnlace"] != true || detalle.body["email"] != "" {
		t.Fatalf("detalle = %v", detalle.body)
	}

	// No aparece en la bandeja de nadie: no tiene email.
	if mias := e.pedir(t, &e.otro, http.MethodGet, "/invitaciones", nil); len(mias.list) != 0 {
		t.Fatalf("mis invitaciones = %v", mias.list)
	}

	// Rechazar un enlace no lo invalida para el resto.
	esperar(t, e.pedir(t, &e.integrante, http.MethodPost, "/invitaciones/"+token+"/rechazar", nil),
		http.StatusNoContent, "")
	if d := e.pedir(t, nil, http.MethodGet, "/invitaciones/"+token, nil); d.body["cancelada"] != false {
		t.Fatalf("el enlace quedó cancelado al rechazarlo: %v", d.body)
	}

	// Quien ya está en el proyecto no lo consume.
	esperar(t, e.pedir(t, &e.integrante, http.MethodPost, "/invitaciones/"+token+"/aceptar", nil),
		http.StatusConflict, "Ya sos integrante de este proyecto")

	// Lo acepta cualquier cuenta, una sola vez.
	esperar(t, e.pedir(t, &e.otro, http.MethodPost, "/invitaciones/"+token+"/aceptar", nil), http.StatusOK, "")
	esperar(t, e.pedir(t, &e.invitado, http.MethodPost, "/invitaciones/"+token+"/aceptar", nil),
		http.StatusConflict, "La invitación ya fue aceptada")
}

// Dos cuentas abren el mismo enlace a la vez: solo una se suma.
func TestInvitaciones_EnlaceAceptadoEnParalelo(t *testing.T) {
	e := prepararEscenarioInvitaciones(t)

	for intento := 0; intento < 5; intento++ {
		creada := e.pedir(t, &e.propietario, http.MethodPost, e.rutaInvitaciones()+"/enlace",
			map[string]any{"codRol": rolMusico})
		esperar(t, creada, http.StatusCreated, "")
		token := tokenDe(t, creada)

		var wg sync.WaitGroup
		largada := make(chan struct{})
		codigos := make([]int, 2)
		for i, u := range []*usuarioInvitacion{&e.invitado, &e.otro} {
			wg.Add(1)
			go func(i int, u *usuarioInvitacion) {
				defer wg.Done()
				<-largada
				codigos[i] = e.pedir(t, u, http.MethodPost, "/invitaciones/"+token+"/aceptar", nil).code
			}(i, u)
		}
		close(largada)
		wg.Wait()

		ok, conflicto := 0, 0
		for _, c := range codigos {
			switch c {
			case http.StatusOK:
				ok++
			case http.StatusConflict:
				conflicto++
			}
		}
		if ok != 1 || conflicto != 1 {
			t.Fatalf("intento %d: códigos %v, quería un 200 y un 409", intento, codigos)
		}

		// Sumados por enlace: todos menos propietario e integrante inicial.
		var sumados int
		if err := e.db.QueryRow(`
			SELECT COUNT(*) FROM integranteproyecto
			WHERE codigoproyecto = $1 AND espropietario = FALSE AND codrol = $2
			  AND fechahorabajaintegranteproy IS NULL
		`, e.proyecto, rolMusico).Scan(&sumados); err != nil {
			t.Fatal(err)
		}
		if sumados != 2 {
			t.Fatalf("intento %d: %d músicos en el proyecto, quería 2 (integrante + 1 por enlace)", intento, sumados)
		}

		// Para el próximo intento, quien entró vuelve a quedar afuera.
		if _, err := e.db.Exec(`
			DELETE FROM integranteproyecto ip USING integrante i
			WHERE ip.codintegrante = i.codintegrante AND ip.codigoproyecto = $1
			  AND i.codigousuario IN ($2, $3)
		`, e.proyecto, e.invitado.codigo, e.otro.codigo); err != nil {
			t.Fatal(err)
		}
	}
}

// Una cuenta acepta a la vez dos invitaciones distintas del mismo proyecto:
// entra una vez, la otra responde 409 y sigue pendiente.
func TestInvitaciones_MismaCuentaDosInvitacionesEnParalelo(t *testing.T) {
	e := prepararEscenarioInvitaciones(t)

	for intento := 0; intento < 5; intento++ {
		porEmail := e.pedir(t, &e.propietario, http.MethodPost, e.rutaInvitaciones(),
			map[string]any{"email": e.otro.email, "codRol": rolMusico})
		esperar(t, porEmail, http.StatusCreated, "")
		porEnlace := e.pedir(t, &e.propietario, http.MethodPost, e.rutaInvitaciones()+"/enlace",
			map[string]any{"codRol": rolProductor})
		esperar(t, porEnlace, http.StatusCreated, "")
		tokens := []string{tokenDe(t, porEmail), tokenDe(t, porEnlace)}

		var wg sync.WaitGroup
		largada := make(chan struct{})
		respuestas := make([]respuestaInvitacion, 2)
		for i, token := range tokens {
			wg.Add(1)
			go func(i int, token string) {
				defer wg.Done()
				<-largada
				respuestas[i] = e.pedir(t, &e.otro, http.MethodPost, "/invitaciones/"+token+"/aceptar", nil)
			}(i, token)
		}
		close(largada)
		wg.Wait()

		perdedora, ganadoras := -1, 0
		for i, r := range respuestas {
			switch r.code {
			case http.StatusOK:
				ganadoras++
			case http.StatusConflict:
				if r.body["error"] != "Ya sos integrante de este proyecto" {
					t.Fatalf("intento %d: 409 con %v", intento, r.body)
				}
				perdedora = i
			default:
				t.Fatalf("intento %d: status %d (%v)", intento, r.code, r.body)
			}
		}
		if ganadoras != 1 || perdedora == -1 {
			t.Fatalf("intento %d: quería un 200 y un 409, fue %d y %d",
				intento, respuestas[0].code, respuestas[1].code)
		}

		// La que perdió no quedó consumida.
		detalle := e.pedir(t, nil, http.MethodGet, "/invitaciones/"+tokens[perdedora], nil)
		if detalle.body["yaAceptada"] != false {
			t.Fatalf("intento %d: la invitación perdedora quedó aceptada: %v", intento, detalle.body)
		}

		if _, err := e.db.Exec(`
			DELETE FROM integranteproyecto ip USING integrante i
			WHERE ip.codintegrante = i.codintegrante AND ip.codigoproyecto = $1 AND i.codigousuario = $2
		`, e.proyecto, e.otro.codigo); err != nil {
			t.Fatal(err)
		}
		if _, err := e.db.Exec(
			`UPDATE invitacionproyecto SET fechahorabajainvitacionproy = CURRENT_TIMESTAMP
			 WHERE codigoproyecto = $1 AND fechahoraaceptacioninvitacion IS NULL`, e.proyecto,
		); err != nil {
			t.Fatal(err)
		}
	}
}
