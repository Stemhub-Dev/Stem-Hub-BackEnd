package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/handler"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/middleware"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/service"
	"github.com/gin-gonic/gin"
)

// Tests de integración del Tablero (HU-DASH-B01/B02) contra PostgreSQL real
// (mismas condiciones y mismo TEST_DATABASE_URL que
// mis_canciones_integracion_test.go). Requieren la migración 023 (catálogo
// de etapas de versionado).

type escenarioTablero struct {
	router *gin.Engine

	usuario          int64
	usuarioSinPerfil int64

	alfa, beta, ajeno, dadoDeBaja int64
}

// prepararEscenarioTablero carga, con fechas relativas a ahora (d = días):
//
//	"Alfa"          miembro; creado hace 90d
//	                  A: v1 hace 50d, v2 hace 10d (Mezcla)
//	                     comentarios en v2: pendiente hace 5d, resuelto hace 3d,
//	                     pendiente dado de baja hace 2d
//	                  B: v1 hace 40d (sin etapa); comentario pendiente hace 45d
//	                  C: dada de baja, v1 hace 5d
//	"Beta"          miembro; creado hace 10d
//	                  D: v1 hace 2d (Maquetación)
//	"Ajeno"         el usuario no participa; E: v1 hace 1d
//	"Dado de baja"  miembro; creado hace 100d, dado de baja hace 1d
//	"Abandonado"    el usuario dejó de participar; F: v1 hace 1d
func prepararEscenarioTablero(t *testing.T) escenarioTablero {
	t.Helper()

	db := conectarBaseDePrueba(t)
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())
	ahora := time.Now()
	hace := func(dias float64) time.Time {
		return ahora.Add(-time.Duration(dias * float64(24*time.Hour)))
	}

	var (
		usuarios    []int64
		integrantes []int64
		proyectos   []int64
		canciones   []int64
		versiones   []int64
		comentarios []int64
	)

	t.Cleanup(func() {
		borrar := func(consulta string, ids []int64) {
			for _, id := range ids {
				if _, err := db.Exec(consulta, id); err != nil {
					t.Errorf("limpieza (%s, %d): %v", consulta, id, err)
				}
			}
		}
		borrar(`DELETE FROM comentario WHERE codigocomentario = $1`, comentarios)
		borrar(`DELETE FROM cancionversion WHERE codigocancionversion = $1`, versiones)
		borrar(`DELETE FROM cancion WHERE codigocancion = $1`, canciones)
		borrar(`DELETE FROM integranteproyecto WHERE codigoproyecto = $1`, proyectos)
		borrar(`DELETE FROM proyecto WHERE codigoproyecto = $1`, proyectos)
		borrar(`DELETE FROM integrante WHERE codintegrante = $1`, integrantes)
		borrar(`DELETE FROM usuario WHERE codigousuario = $1`, usuarios)
	})

	debe := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("carga del escenario: %v", err)
		}
	}

	nuevoUsuario := func(nombre string) int64 {
		var id int64
		debe(db.QueryRow(
			`INSERT INTO usuario (email, idautenticacion)
			 VALUES ($1, gen_random_uuid()) RETURNING codigousuario`,
			nombre+"-"+sufijo+"@test.local",
		).Scan(&id))
		usuarios = append(usuarios, id)
		return id
	}

	nuevoIntegrante := func(codigoUsuario int64) int64 {
		var id int64
		debe(db.QueryRow(
			`INSERT INTO integrante (codigousuario, nombreintegrante)
			 VALUES ($1, $2) RETURNING codintegrante`,
			codigoUsuario, "integrante-"+sufijo,
		).Scan(&id))
		integrantes = append(integrantes, id)
		return id
	}

	nuevoProyecto := func(nombre string, baja *time.Time) int64 {
		var id int64
		debe(db.QueryRow(
			`INSERT INTO proyecto (nombreproyecto, codtipoproy, codestadoproy, fechahorabajaproyecto)
			 VALUES ($1, $2, $3, $4) RETURNING codigoproyecto`,
			nombre, tipoSingle, estadoEnDesarrollo, baja,
		).Scan(&id))
		proyectos = append(proyectos, id)
		return id
	}

	nuevaMembresia := func(codIntegrante, codProyecto int64, alta time.Time, baja *time.Time) {
		_, err := db.Exec(
			`INSERT INTO integranteproyecto
			   (codintegrante, codigoproyecto, codrol, espropietario, fechahoraaltaintegranteproy, fechahorabajaintegranteproy)
			 VALUES ($1, $2, $3, TRUE, $4, $5)`,
			codIntegrante, codProyecto, rolProductor, alta, baja,
		)
		debe(err)
	}

	nuevaCancion := func(codProyecto int64, nombre string, baja *time.Time) int64 {
		var id int64
		debe(db.QueryRow(
			`INSERT INTO cancion (codigoproyecto, nombrecancion, fechahorabajacancion)
			 VALUES ($1, $2, $3) RETURNING codigocancion`,
			codProyecto, nombre, baja,
		).Scan(&id))
		canciones = append(canciones, id)
		return id
	}

	nuevaVersion := func(codCancion int64, numero int, alta time.Time, etapa string) int64 {
		var id int64
		debe(db.QueryRow(
			`INSERT INTO cancionversion (codigocancion, numeroversion, fechahoraaltaversion, formatoarchivocancionver, codetapaversion)
			 VALUES ($1, $2, $3, 'mp3',
			   (SELECT codetapaversion FROM etapaversionado WHERE nombreetapaversion = NULLIF($4, '')))
			 RETURNING codigocancionversion`,
			codCancion, numero, alta, etapa,
		).Scan(&id))
		versiones = append(versiones, id)
		return id
	}

	nuevoComentario := func(codIntegrante, codVersion int64, estado string, alta time.Time, baja *time.Time) {
		var id int64
		debe(db.QueryRow(
			`INSERT INTO comentario (codintegrante, codestadocom, codigocancionversion, descripcioncomentario,
			   fechahoraaltacomentario, fechahorabajacomentario)
			 SELECT $1, ec.codestadocom, $2, 'comentario', $4, $5
			 FROM estadocomentario ec WHERE ec.nombreestadocom = $3
			 RETURNING codigocomentario`,
			codIntegrante, codVersion, estado, alta, baja,
		).Scan(&id))
		comentarios = append(comentarios, id)
	}

	yo := nuevoUsuario("tablero-yo")
	otro := nuevoUsuario("tablero-otro")
	sinPerfil := nuevoUsuario("tablero-sin-perfil")

	integranteYo := nuevoIntegrante(yo)
	integranteOtro := nuevoIntegrante(otro)

	bajaProyecto := hace(1)
	ahoraBaja := ahora

	alfa := nuevoProyecto("Alfa "+sufijo, nil)
	beta := nuevoProyecto("Beta "+sufijo, nil)
	ajeno := nuevoProyecto("Ajeno "+sufijo, nil)
	dadoDeBaja := nuevoProyecto("Dado de baja "+sufijo, &bajaProyecto)
	abandonado := nuevoProyecto("Abandonado "+sufijo, nil)

	nuevaMembresia(integranteYo, alfa, hace(90), nil)
	nuevaMembresia(integranteYo, beta, hace(10), nil)
	nuevaMembresia(integranteOtro, ajeno, hace(90), nil)
	nuevaMembresia(integranteYo, dadoDeBaja, hace(100), nil)
	nuevaMembresia(integranteYo, abandonado, hace(90), &ahoraBaja)

	cancionA := nuevaCancion(alfa, "A", nil)
	nuevaVersion(cancionA, 1, hace(50), "")
	versionA2 := nuevaVersion(cancionA, 2, hace(10), "Mezcla")
	nuevoComentario(integranteYo, versionA2, "Pendiente", hace(5), nil)
	nuevoComentario(integranteYo, versionA2, "Resuelto", hace(3), nil)
	nuevoComentario(integranteYo, versionA2, "Pendiente", hace(2), &ahoraBaja)

	cancionB := nuevaCancion(alfa, "B", nil)
	versionB1 := nuevaVersion(cancionB, 1, hace(40), "")
	nuevoComentario(integranteYo, versionB1, "Pendiente", hace(45), nil)

	cancionC := nuevaCancion(alfa, "C", &ahoraBaja)
	nuevaVersion(cancionC, 1, hace(5), "")

	cancionD := nuevaCancion(beta, "D", nil)
	nuevaVersion(cancionD, 1, hace(2), "Maquetación")

	nuevaVersion(nuevaCancion(ajeno, "E", nil), 1, hace(1), "Mezcla")
	nuevaVersion(nuevaCancion(abandonado, "F", nil), 1, hace(1), "Mezcla")

	gin.SetMode(gin.TestMode)

	h := handler.NewTableroHandler(service.NewTableroService(
		repository.NewTableroRepository(db),
		repository.NewProyectoRepository(db),
		repository.NewIntegranteRepository(db),
	))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		var codigoUsuario int64
		fmt.Sscan(c.GetHeader("X-Usuario"), &codigoUsuario)
		c.Set(middleware.UsuarioContextKey, &model.Usuario{CodigoUsuario: codigoUsuario})
		c.Next()
	})
	router.GET("/tablero/indicadores", h.ObtenerIndicadores)
	router.GET("/tablero/graficos/versiones-por-proyecto", h.ObtenerVersionesPorProyecto)
	router.GET("/tablero/graficos/canciones-por-etapa", h.ObtenerCancionesPorEtapa)
	router.GET("/tablero/graficos/actividad", h.ObtenerActividad)

	return escenarioTablero{
		router:           router,
		usuario:          yo,
		usuarioSinPerfil: sinPerfil,
		alfa:             alfa,
		beta:             beta,
		ajeno:            ajeno,
		dadoDeBaja:       dadoDeBaja,
	}
}

func (e escenarioTablero) pedir(t *testing.T, codigoUsuario int64, url string, destino any) int {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, url, nil)
	request.Header.Set("X-Usuario", fmt.Sprint(codigoUsuario))

	grabador := httptest.NewRecorder()
	e.router.ServeHTTP(grabador, request)

	if grabador.Code == http.StatusOK && destino != nil {
		if err := json.Unmarshal(grabador.Body.Bytes(), destino); err != nil {
			t.Fatalf("respuesta inválida (%s): %v", grabador.Body.String(), err)
		}
	}

	return grabador.Code
}

func TestIntegracionTablero(t *testing.T) {
	e := prepararEscenarioTablero(t)

	t.Run("indicadores globales", func(t *testing.T) {
		var indicadores dto.TableroIndicadoresResponse
		if status := e.pedir(t, e.usuario, "/tablero/indicadores", &indicadores); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}

		esperado := dto.TableroIndicadoresResponse{
			// Hoy: Alfa y Beta. Hace 30d: Alfa y "Dado de baja".
			TotalProyectosActivos: dto.IndicadorEnteroResponse{Valor: 2, Variacion: 0},
			// Hoy: A, B, D. Hace 30d: A, B.
			TotalCanciones: dto.IndicadorEnteroResponse{Valor: 3, Variacion: 1},
			// Últimos 30d: A v2, D v1. Los 30d anteriores: A v1, B v1.
			VersionesUltimos30Dias: dto.IndicadorEnteroResponse{Valor: 2, Variacion: 0},
			// Hoy: los de A v2 (5d) y B v1 (45d). Hace 30d: solo el de B v1.
			ComentariosPendientes: dto.IndicadorEnteroResponse{Valor: 2, Variacion: 1},
			// Hoy 4 versiones / 3 canciones; hace 30d 2 / 2.
			PromedioVersionesPorCancion: dto.IndicadorDecimalResponse{Valor: 1.33, Variacion: 0.33},
		}
		if indicadores != esperado {
			t.Errorf("indicadores = %+v\nse esperaba   %+v", indicadores, esperado)
		}
	})

	t.Run("indicadores de un proyecto", func(t *testing.T) {
		var indicadores dto.TableroIndicadoresResponse
		url := fmt.Sprintf("/tablero/indicadores?proyectoId=%d", e.beta)
		if status := e.pedir(t, e.usuario, url, &indicadores); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}

		esperado := dto.TableroIndicadoresResponse{
			TotalProyectosActivos:       dto.IndicadorEnteroResponse{Valor: 1, Variacion: 1},
			TotalCanciones:              dto.IndicadorEnteroResponse{Valor: 1, Variacion: 1},
			VersionesUltimos30Dias:      dto.IndicadorEnteroResponse{Valor: 1, Variacion: 1},
			ComentariosPendientes:       dto.IndicadorEnteroResponse{},
			PromedioVersionesPorCancion: dto.IndicadorDecimalResponse{Valor: 1, Variacion: 1},
		}
		if indicadores != esperado {
			t.Errorf("indicadores = %+v\nse esperaba   %+v", indicadores, esperado)
		}
	})

	t.Run("usuario sin perfil ve todo en 0", func(t *testing.T) {
		var indicadores dto.TableroIndicadoresResponse
		if status := e.pedir(t, e.usuarioSinPerfil, "/tablero/indicadores", &indicadores); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}
		if indicadores != (dto.TableroIndicadoresResponse{}) {
			t.Errorf("indicadores = %+v, se esperaba todo en 0", indicadores)
		}

		for _, url := range []string{
			"/tablero/graficos/versiones-por-proyecto",
			"/tablero/graficos/canciones-por-etapa",
			"/tablero/graficos/actividad",
		} {
			var datos []any
			if status := e.pedir(t, e.usuarioSinPerfil, url, &datos); status != http.StatusOK || datos == nil || len(datos) != 0 {
				t.Errorf("%s: status %d, datos %v; se esperaba 200 y []", url, status, datos)
			}
		}
	})

	t.Run("versiones por proyecto", func(t *testing.T) {
		var datos []dto.TableroVersionesPorProyectoResponse
		if status := e.pedir(t, e.usuario, "/tablero/graficos/versiones-por-proyecto", &datos); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}

		if len(datos) != 2 ||
			datos[0].ProyectoID != e.alfa || datos[0].CantidadVersiones != 3 ||
			datos[1].ProyectoID != e.beta || datos[1].CantidadVersiones != 1 {
			t.Errorf("datos = %+v; se esperaba Alfa (3) y Beta (1)", datos)
		}
	})

	t.Run("canciones por etapa", func(t *testing.T) {
		var datos []dto.TableroCancionesPorEtapaResponse
		if status := e.pedir(t, e.usuario, "/tablero/graficos/canciones-por-etapa", &datos); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}

		esperado := []dto.TableroCancionesPorEtapaResponse{
			{Etapa: "Maquetación", Cantidad: 1},
			{Etapa: "Composición", Cantidad: 0},
			{Etapa: "Mezcla", Cantidad: 1},
			{Etapa: service.EtapaSinAsignar, Cantidad: 1},
		}
		if !reflect.DeepEqual(datos, esperado) {
			t.Errorf("datos = %+v\nse esperaba %+v", datos, esperado)
		}
	})

	t.Run("actividad", func(t *testing.T) {
		for agrupacion, periodos := range map[string][2]int{
			"semanal": {13, 15},
			"mensual": {4, 4},
		} {
			var datos []dto.TableroActividadResponse
			url := "/tablero/graficos/actividad?agrupacion=" + agrupacion
			if status := e.pedir(t, e.usuario, url, &datos); status != http.StatusOK {
				t.Fatalf("%s: status = %d", agrupacion, status)
			}

			if len(datos) < periodos[0] || len(datos) > periodos[1] {
				t.Errorf("%s: %d períodos, se esperaban entre %d y %d", agrupacion, len(datos), periodos[0], periodos[1])
			}

			var versiones, comentarios int64
			for _, periodo := range datos {
				versiones += periodo.Versiones
				comentarios += periodo.Comentarios
			}

			// Versiones: A v1, A v2, B v1, D v1. Comentarios: los 3 sin baja.
			if versiones != 4 || comentarios != 3 {
				t.Errorf("%s: %d versiones y %d comentarios, se esperaban 4 y 3", agrupacion, versiones, comentarios)
			}
		}
	})

	t.Run("errores por proyecto", func(t *testing.T) {
		casos := []struct {
			nombre  string
			usuario int64
			query   string
			status  int
		}{
			{"proyecto ajeno es 403", e.usuario, fmt.Sprintf("proyectoId=%d", e.ajeno), http.StatusForbidden},
			{"proyecto dado de baja es 404", e.usuario, fmt.Sprintf("proyectoId=%d", e.dadoDeBaja), http.StatusNotFound},
			{"proyecto inexistente es 404", e.usuario, "proyectoId=999999999", http.StatusNotFound},
			{"sin perfil con proyecto es 403", e.usuarioSinPerfil, fmt.Sprintf("proyectoId=%d", e.alfa), http.StatusForbidden},
			{"proyectoId inválido es 400", e.usuario, "proyectoId=abc", http.StatusBadRequest},
		}

		for _, caso := range casos {
			for _, ruta := range []string{
				"/tablero/indicadores",
				"/tablero/graficos/versiones-por-proyecto",
				"/tablero/graficos/canciones-por-etapa",
				"/tablero/graficos/actividad",
			} {
				if status := e.pedir(t, caso.usuario, ruta+"?"+caso.query, nil); status != caso.status {
					t.Errorf("%s (%s): status = %d, se esperaba %d", caso.nombre, ruta, status, caso.status)
				}
			}
		}
	})
}
