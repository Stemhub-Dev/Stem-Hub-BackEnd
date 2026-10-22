package handler_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// Tests de integración de lo que usa el Inicio: novedades, indicador de
// stems, canciones listas por proyecto y registro de la visita. Mismas
// condiciones que tablero_integracion_test.go; requieren hasta la
// migración 027.

type escenarioInicio struct {
	db     *sql.DB
	router *gin.Engine
	ahora  time.Time

	usuario, usuarioOtro int64
	integrante           int64
	proyecto             int64
}

// prepararEscenarioInicio carga, con fechas relativas a ahora (d = días):
//
//	"Compartido"  yo y otro somos integrantes
//	                A: v1 de otro hace 3d (última etapa del catálogo)
//	                   comentario de otro hace 1d, con respuesta de otro hace 0,5d
//	                   comentario mío hace 0,4d
//	                   2 stems de otro hace 0,2d; 1 stem con IA mío
//	                   separación mía (4 stems) terminada hace 0,1d
//	                B: v1 mía hace 2d (Maquetación)
//	"Ajeno"       solo un extraño; G: v1 del extraño hace 0,05d
func prepararEscenarioInicio(t *testing.T) escenarioInicio {
	t.Helper()

	db := conectarBaseDePrueba(t)
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())
	ahora := time.Now()
	hace := func(dias float64) time.Time {
		return ahora.Add(-time.Duration(dias * float64(24*time.Hour)))
	}

	var usuarios, integrantes, proyectos, canciones, versiones, comentarios []int64

	t.Cleanup(func() {
		borrar := func(consulta string, ids []int64) {
			for _, id := range ids {
				if _, err := db.Exec(consulta, id); err != nil {
					t.Errorf("limpieza (%s, %d): %v", consulta, id, err)
				}
			}
		}
		borrar(`DELETE FROM comentariorespuesta WHERE codigocomentario = $1`, comentarios)
		borrar(`DELETE FROM comentario WHERE codigocomentario = $1`, comentarios)
		borrar(`DELETE FROM stem WHERE codigocancionversion = $1`, versiones)
		borrar(`DELETE FROM separacionstem WHERE codigocancionversion = $1`, versiones)
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

	insertar := func(destino *[]int64, consulta string, args ...any) int64 {
		var id int64
		debe(db.QueryRow(consulta, args...).Scan(&id))
		if destino != nil {
			*destino = append(*destino, id)
		}
		return id
	}

	nuevoIntegrante := func(nombre string) (int64, int64) {
		usuario := insertar(&usuarios,
			`INSERT INTO usuario (email, idautenticacion) VALUES ($1, gen_random_uuid()) RETURNING codigousuario`,
			nombre+"-"+sufijo+"@test.local")
		integrante := insertar(&integrantes,
			`INSERT INTO integrante (codigousuario, nombreintegrante) VALUES ($1, $2) RETURNING codintegrante`,
			usuario, nombre)
		return usuario, integrante
	}

	nuevoProyecto := func(nombre string, miembros ...int64) int64 {
		proyecto := insertar(&proyectos,
			`INSERT INTO proyecto (nombreproyecto, codtipoproy, codestadoproy) VALUES ($1, $2, $3) RETURNING codigoproyecto`,
			nombre+" "+sufijo, tipoSingle, estadoEnDesarrollo)
		for i, miembro := range miembros {
			_, err := db.Exec(
				`INSERT INTO integranteproyecto (codintegrante, codigoproyecto, codrol, espropietario, fechahoraaltaintegranteproy)
				 VALUES ($1, $2, $3, $4, $5)`,
				miembro, proyecto, rolProductor, i == 0, hace(30))
			debe(err)
		}
		return proyecto
	}

	nuevaVersion := func(proyecto int64, cancion string, autor int64, alta time.Time, etapa string) int64 {
		codCancion := insertar(&canciones,
			`INSERT INTO cancion (codigoproyecto, nombrecancion) VALUES ($1, $2) RETURNING codigocancion`,
			proyecto, cancion)
		return insertar(&versiones,
			`INSERT INTO cancionversion (codigocancion, numeroversion, fechahoraaltaversion, formatoarchivocancionver,
			   codetapaversion, codintegrantealtaversion)
			 VALUES ($1, 1, $2, 'mp3', (SELECT codetapaversion FROM etapaversionado WHERE nombreetapaversion = $3), $4)
			 RETURNING codigocancionversion`,
			codCancion, alta, etapa, autor)
	}

	nuevoComentario := func(autor, version int64, alta time.Time) int64 {
		return insertar(&comentarios,
			`INSERT INTO comentario (codintegrante, codestadocom, codigocancionversion, descripcioncomentario, fechahoraaltacomentario)
			 VALUES ($1, 1, $2, 'comentario', $3) RETURNING codigocomentario`,
			autor, version, alta)
	}

	nuevoStem := func(version int64, nombre string, autor int64, alta time.Time, conIA bool) {
		_, err := db.Exec(
			`INSERT INTO stem (codigocancionversion, nombrestem, codcategoriastem, urlarchivostem, formatoarchivostem,
			   nombrearchivostem, generadoconia, codintegrantealtastem, fechahoraaltastem)
			 VALUES ($1, $2, (SELECT MIN(codcategoriastem) FROM categoriastem), 'k', 'wav', 'a.wav', $3, $4, $5)`,
			version, nombre, conIA, autor, alta)
		debe(err)
	}

	var ultimaEtapa string
	debe(db.QueryRow(
		`SELECT nombreetapaversion FROM etapaversionado WHERE fechahorabajaetapaversion IS NULL
		 ORDER BY ordenetapaversion DESC, codetapaversion DESC LIMIT 1`,
	).Scan(&ultimaEtapa))

	yo, integranteYo := nuevoIntegrante("inicio-yo")
	otro, integranteOtro := nuevoIntegrante("inicio-otro")
	_, integranteExtrano := nuevoIntegrante("inicio-extrano")

	compartido := nuevoProyecto("Compartido", integranteYo, integranteOtro)
	ajeno := nuevoProyecto("Ajeno", integranteExtrano)

	versionA := nuevaVersion(compartido, "A", integranteOtro, hace(3), ultimaEtapa)
	nuevaVersion(compartido, "B", integranteYo, hace(2), "Maquetación")

	comentario := nuevoComentario(integranteOtro, versionA, hace(1))
	_, err := db.Exec(
		`INSERT INTO comentariorespuesta (codigocomentario, codintegrante, descripcionrespuesta, fechahoraaltarespuesta)
		 VALUES ($1, $2, 'respuesta', $3)`,
		comentario, integranteOtro, hace(0.5))
	debe(err)
	nuevoComentario(integranteYo, versionA, hace(0.4))

	nuevoStem(versionA, "Voz", integranteOtro, hace(0.2), false)
	nuevoStem(versionA, "Coros", integranteOtro, hace(0.2), false)
	nuevoStem(versionA, "Bajo IA", integranteYo, hace(0.1), true)
	_, err = db.Exec(
		`INSERT INTO separacionstem (codigocancionversion, codintegrante, cantidadstems, estadoseparacion,
		   fechahorasolicitud, fechahorafin)
		 VALUES ($1, $2, 4, 'COMPLETADA', $3, $4)`,
		versionA, integranteYo, hace(0.11), hace(0.1))
	debe(err)

	nuevaVersion(ajeno, "G", integranteExtrano, hace(0.05), "Mezcla")

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
	router.GET("/tablero/novedades", h.ObtenerNovedades)
	router.POST("/tablero/visita", h.RegistrarVisita)

	return escenarioInicio{
		db:          db,
		router:      router,
		ahora:       ahora,
		usuario:     yo,
		usuarioOtro: otro,
		integrante:  integranteYo,
		proyecto:    compartido,
	}
}

func (e escenarioInicio) pedir(t *testing.T, metodo string, codigoUsuario int64, ruta string, destino any) int {
	t.Helper()

	request := httptest.NewRequest(metodo, ruta, nil)
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

func TestIntegracionInicio(t *testing.T) {
	e := prepararEscenarioInicio(t)

	t.Run("novedades de los demás, sin lo propio salvo separaciones", func(t *testing.T) {
		var novedades dto.TableroNovedadesResponse
		desde := url.QueryEscape(e.ahora.Add(-18 * time.Hour).Format(time.RFC3339))
		if status := e.pedir(t, http.MethodGet, e.usuario, "/tablero/novedades?desde="+desde, &novedades); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}

		esperado := []struct {
			tipo     string
			cancion  string
			cantidad int64
		}{
			{dto.NovedadSeparacion, "A", 4},
			{dto.NovedadStems, "A", 2},
			{dto.NovedadRespuesta, "A", 1},
			{dto.NovedadComentario, "A", 1},
			{dto.NovedadVersion, "A", 1},
		}
		if len(novedades.Items) != len(esperado) {
			t.Fatalf("novedades = %+v, se esperaban %d", novedades.Items, len(esperado))
		}
		for i, item := range novedades.Items {
			if item.Tipo != esperado[i].tipo || item.NombreCancion != esperado[i].cancion ||
				item.Cantidad != esperado[i].cantidad || item.CodigoProyecto != e.proyecto {
				t.Errorf("novedad %d = %+v, se esperaba %+v", i, item, esperado[i])
			}
		}
		// Separación (0,1d), stems (0,2d) y respuesta (0,5d) son posteriores a hace 18 h.
		if novedades.NovedadesDesde != 3 {
			t.Errorf("novedadesDesde = %d, se esperaba 3", novedades.NovedadesDesde)
		}
	})

	t.Run("el límite no cambia el total desde", func(t *testing.T) {
		var novedades dto.TableroNovedadesResponse
		desde := url.QueryEscape(e.ahora.Add(-18 * time.Hour).Format(time.RFC3339))
		if status := e.pedir(t, http.MethodGet, e.usuario, "/tablero/novedades?limite=2&desde="+desde, &novedades); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}
		if len(novedades.Items) != 2 || novedades.NovedadesDesde != 3 {
			t.Errorf("items = %d, novedadesDesde = %d; se esperaban 2 y 3", len(novedades.Items), novedades.NovedadesDesde)
		}
	})

	t.Run("indicador de stems", func(t *testing.T) {
		var indicadores dto.TableroIndicadoresResponse
		if status := e.pedir(t, http.MethodGet, e.usuario, "/tablero/indicadores", &indicadores); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}
		// Los 3 stems de A, todos subidos dentro de los últimos 30 días.
		if indicadores.TotalStems != (dto.IndicadorEnteroResponse{Valor: 3, Variacion: 3}) {
			t.Errorf("totalStems = %+v", indicadores.TotalStems)
		}
	})

	t.Run("canciones listas del proyecto", func(t *testing.T) {
		proyectos, err := repository.NewProyectoRepository(e.db).ListarPorIntegrante(
			e.integrante,
			dto.ListarProyectosFiltro{Pagina: 1, TamanoPagina: 10},
			0,
		)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(proyectos) != 1 || proyectos[0].CantidadCanciones != 2 || proyectos[0].CancionesListas != 1 {
			t.Errorf("proyectos = %+v, se esperaba Compartido con 1 de 2 canciones listas", proyectos)
		}
	})

	t.Run("registrar visita devuelve la anterior", func(t *testing.T) {
		var primera, segunda dto.TableroVisitaResponse
		if status := e.pedir(t, http.MethodPost, e.usuarioOtro, "/tablero/visita", &primera); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}
		if primera.VisitaAnterior != nil {
			t.Errorf("primera visita: anterior = %v, se esperaba nil", primera.VisitaAnterior)
		}

		if status := e.pedir(t, http.MethodPost, e.usuarioOtro, "/tablero/visita", &segunda); status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}
		if segunda.VisitaAnterior == nil || time.Since(*segunda.VisitaAnterior) > time.Minute {
			t.Errorf("segunda visita: anterior = %v, se esperaba la de recién", segunda.VisitaAnterior)
		}
	})
}
