package handler_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Tests de integración de GET /canciones contra un PostgreSQL real con las
// migraciones aplicadas. Recorren handler → service → repository → SQL, que
// es donde un mock no puede detectar errores (sintaxis, tipos de parámetros,
// orden real). Se saltean si no está definida TEST_DATABASE_URL; en CI la
// provee el job con el servicio de Postgres.
//
// Cada ejecución crea sus propios usuarios, proyectos y canciones con un
// sufijo único y los borra al terminar. Como las consultas se acotan al
// integrante, otros datos que haya en la base no interfieren.

type escenarioMisCanciones struct {
	db *sql.DB

	usuarioConPerfil int64
	usuarioSinPerfil int64

	proyectoA     int64
	proyectoB     int64
	proyectoAjeno int64
}

func conectarBaseDePrueba(t *testing.T) *sql.DB {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida: se omiten los tests de integración")
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("no se pudo abrir la base: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("no se pudo conectar a la base: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	return db
}

// prepararEscenario carga:
//
//	Proyecto A (miembro):   "Balada Roja" (v1 hace 3d, v2 hace 1d), "Rock 100%" (hace 2d),
//	                        "mi_tema" (sin versión), "Balada borrada" (dada de baja)
//	Proyecto B (miembro):   "BALADA azul" (hace 5h), "Otra" (hace 10d)
//	Proyecto ajeno:         "Balada ajena" (el usuario no participa)
//	Proyecto dado de baja:  "Balada de proyecto dado de baja"
//	Proyecto abandonado:    "Balada de proyecto abandonado" (membresía dada de baja)
//
// Visibles para el usuario, de la más reciente a la más vieja:
// BALADA azul, Balada Roja, Rock 100%, Otra, mi_tema.
func prepararEscenario(t *testing.T) escenarioMisCanciones {
	t.Helper()

	db := conectarBaseDePrueba(t)
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())

	var (
		usuarios      []int64
		integrantes   []int64
		proyectos     []int64
		canciones     []int64
		membresias    []int64
		codEstado     int64
		codTipo       int64
		codRol        int64
		integranteYo  int64
		integranteOtr int64
	)

	// Se registra primero para que limpie aunque la carga falle a mitad.
	t.Cleanup(func() {
		borrar := func(consulta string, ids []int64) {
			for _, id := range ids {
				if _, err := db.Exec(consulta, id); err != nil {
					t.Errorf("limpieza (%s, %d): %v", consulta, id, err)
				}
			}
		}
		borrar(`DELETE FROM cancionversion WHERE codigocancion = $1`, canciones)
		borrar(`DELETE FROM cancion WHERE codigocancion = $1`, canciones)
		borrar(`DELETE FROM integranteproyecto WHERE codigointegranteproyecto = $1`, membresias)
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

	debe(db.QueryRow(`SELECT MIN(codestadoproy) FROM estadoproyecto`).Scan(&codEstado))
	debe(db.QueryRow(`SELECT MIN(codtipoproy) FROM tipoproyecto`).Scan(&codTipo))
	debe(db.QueryRow(`SELECT MIN(codrol) FROM rol`).Scan(&codRol))

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

	nuevoProyecto := func(nombre string, dadoDeBaja bool) int64 {
		var baja *time.Time
		if dadoDeBaja {
			ahora := time.Now()
			baja = &ahora
		}
		var id int64
		debe(db.QueryRow(
			`INSERT INTO proyecto (nombreproyecto, codestadoproy, codtipoproy, fechahorabajaproyecto)
			 VALUES ($1, $2, $3, $4) RETURNING codigoproyecto`,
			nombre+" "+sufijo, codEstado, codTipo, baja,
		).Scan(&id))
		proyectos = append(proyectos, id)
		return id
	}

	nuevaMembresia := func(codIntegrante, codProyecto int64, dadaDeBaja bool) {
		var baja *time.Time
		if dadaDeBaja {
			ahora := time.Now()
			baja = &ahora
		}
		var id int64
		debe(db.QueryRow(
			`INSERT INTO integranteproyecto (codintegrante, codigoproyecto, codrol, fechahorabajaintegranteproy)
			 VALUES ($1, $2, $3, $4) RETURNING codigointegranteproyecto`,
			codIntegrante, codProyecto, codRol, baja,
		).Scan(&id))
		membresias = append(membresias, id)
	}

	nuevaCancion := func(codProyecto int64, nombre string, dadaDeBaja bool, haceVersiones ...time.Duration) {
		var baja *time.Time
		if dadaDeBaja {
			ahora := time.Now()
			baja = &ahora
		}
		var id int64
		debe(db.QueryRow(
			`INSERT INTO cancion (codigoproyecto, nombrecancion, fechahorabajacancion)
			 VALUES ($1, $2, $3) RETURNING codigocancion`,
			codProyecto, nombre, baja,
		).Scan(&id))
		canciones = append(canciones, id)

		for i, hace := range haceVersiones {
			_, err := db.Exec(
				`INSERT INTO cancionversion (codigocancion, numeroversion, fechahoraaltaversion, formatoarchivocancionver)
				 VALUES ($1, $2, $3, 'mp3')`,
				id, i+1, time.Now().Add(-hace),
			)
			debe(err)
		}
	}

	yo := nuevoUsuario("yo")
	otro := nuevoUsuario("otro")
	sinPerfil := nuevoUsuario("sin-perfil")

	integranteYo = nuevoIntegrante(yo)
	integranteOtr = nuevoIntegrante(otro)

	proyectoA := nuevoProyecto("A", false)
	proyectoB := nuevoProyecto("B", false)
	proyectoAjeno := nuevoProyecto("Ajeno", false)
	proyectoBaja := nuevoProyecto("Baja", true)
	proyectoAbandonado := nuevoProyecto("Abandonado", false)

	nuevaMembresia(integranteYo, proyectoA, false)
	nuevaMembresia(integranteYo, proyectoB, false)
	nuevaMembresia(integranteYo, proyectoBaja, false)
	nuevaMembresia(integranteYo, proyectoAbandonado, true)
	nuevaMembresia(integranteOtr, proyectoAjeno, false)

	dia := 24 * time.Hour
	nuevaCancion(proyectoA, "Balada Roja", false, 3*dia, 1*dia)
	nuevaCancion(proyectoA, "Rock 100%", false, 2*dia)
	nuevaCancion(proyectoA, "mi_tema", false)
	nuevaCancion(proyectoA, "Balada borrada", true, time.Hour)
	nuevaCancion(proyectoB, "BALADA azul", false, 5*time.Hour)
	nuevaCancion(proyectoB, "Otra", false, 10*dia)
	nuevaCancion(proyectoAjeno, "Balada ajena", false, time.Hour)
	nuevaCancion(proyectoBaja, "Balada de proyecto dado de baja", false, time.Hour)
	nuevaCancion(proyectoAbandonado, "Balada de proyecto abandonado", false, time.Hour)

	return escenarioMisCanciones{
		db:               db,
		usuarioConPerfil: yo,
		usuarioSinPerfil: sinPerfil,
		proyectoA:        proyectoA,
		proyectoB:        proyectoB,
		proyectoAjeno:    proyectoAjeno,
	}
}

func (e escenarioMisCanciones) pedir(
	t *testing.T,
	codigoUsuario int64,
	url string,
) (int, dto.MisCancionesPaginadasResponse) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	cancionHandler := handler.NewCancionHandler(
		service.NewCancionService(
			repository.NewCancionRepository(e.db),
			repository.NewProyectoRepository(e.db),
			repository.NewIntegranteRepository(e.db),
			nil,
		),
	)

	router := gin.New()
	router.GET("/canciones", func(c *gin.Context) {
		c.Set(middleware.UsuarioContextKey, &model.Usuario{CodigoUsuario: codigoUsuario})
		c.Next()
	}, cancionHandler.ListarMisCanciones)

	grabador := httptest.NewRecorder()
	router.ServeHTTP(grabador, httptest.NewRequest(http.MethodGet, url, nil))

	var cuerpo dto.MisCancionesPaginadasResponse
	if grabador.Code == http.StatusOK {
		if err := json.Unmarshal(grabador.Body.Bytes(), &cuerpo); err != nil {
			t.Fatalf("JSON inválido: %v (%s)", err, grabador.Body.String())
		}
		if cuerpo.Data == nil {
			t.Fatalf("data llegó como null: %s", grabador.Body.String())
		}
	}

	return grabador.Code, cuerpo
}

func nombres(respuesta dto.MisCancionesPaginadasResponse) []string {
	resultado := make([]string, 0, len(respuesta.Data))
	for _, cancion := range respuesta.Data {
		resultado = append(resultado, cancion.Nombre)
	}
	return resultado
}

func verificar(
	t *testing.T,
	status int,
	respuesta dto.MisCancionesPaginadasResponse,
	esperados []string,
	totalItems, totalPages, currentPage int,
) {
	t.Helper()

	if status != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", status)
	}
	if obtenidos := nombres(respuesta); !reflect.DeepEqual(obtenidos, esperados) {
		t.Errorf("canciones = %q, se esperaba %q", obtenidos, esperados)
	}
	if respuesta.TotalItems != totalItems ||
		respuesta.TotalPages != totalPages ||
		respuesta.CurrentPage != currentPage {
		t.Errorf(
			"totalItems/totalPages/currentPage = %d/%d/%d, se esperaba %d/%d/%d",
			respuesta.TotalItems, respuesta.TotalPages, respuesta.CurrentPage,
			totalItems, totalPages, currentPage,
		)
	}
}

func TestIntegracionMisCanciones(t *testing.T) {
	e := prepararEscenario(t)
	yo := e.usuarioConPerfil

	t.Run("sin parámetros: defaults, solo canciones propias activas, orden reciente", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/canciones")
		verificar(t, status, respuesta,
			[]string{"BALADA azul", "Balada Roja", "Rock 100%", "Otra", "mi_tema"},
			5, 1, 1)

		primera := respuesta.Data[0]
		if primera.CodigoProyecto != e.proyectoB || primera.NombreProyecto == "" {
			t.Errorf("proyecto de la canción mal mapeado: %+v", primera)
		}
		if primera.VersionActual == nil || primera.VersionActual.NumeroVersion != 1 {
			t.Errorf("versión actual mal mapeada: %+v", primera.VersionActual)
		}
		if respuesta.Data[1].VersionActual == nil || respuesta.Data[1].VersionActual.NumeroVersion != 2 {
			t.Errorf("la versión actual debe ser la más alta: %+v", respuesta.Data[1].VersionActual)
		}
		if respuesta.Data[4].VersionActual != nil {
			t.Errorf("una canción sin versiones no debe tener versión actual")
		}
	})

	t.Run("búsqueda parcial sin distinguir mayúsculas", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/canciones?q=bAlAdA")
		verificar(t, status, respuesta, []string{"BALADA azul", "Balada Roja"}, 2, 1, 1)
	})

	t.Run("búsqueda con comodines se toma literal", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/canciones?q=%25")
		verificar(t, status, respuesta, []string{"Rock 100%"}, 1, 1, 1)

		status, respuesta = e.pedir(t, yo, "/canciones?q=_")
		verificar(t, status, respuesta, []string{"mi_tema"}, 1, 1, 1)
	})

	t.Run("búsqueda sin coincidencias", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/canciones?q=inexistente")
		verificar(t, status, respuesta, []string{}, 0, 0, 1)
	})

	t.Run("filtro por un proyecto", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, fmt.Sprintf("/canciones?proyectoId=%d", e.proyectoB))
		verificar(t, status, respuesta, []string{"BALADA azul", "Otra"}, 2, 1, 1)
	})

	t.Run("filtro por varios proyectos", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, fmt.Sprintf(
			"/canciones?proyectoId=%d&proyectoId=%d", e.proyectoA, e.proyectoB))
		verificar(t, status, respuesta,
			[]string{"BALADA azul", "Balada Roja", "Rock 100%", "Otra", "mi_tema"},
			5, 1, 1)
	})

	t.Run("filtro por proyecto ajeno no expone canciones", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, fmt.Sprintf("/canciones?proyectoId=%d", e.proyectoAjeno))
		verificar(t, status, respuesta, []string{}, 0, 0, 1)
	})

	t.Run("orden por nombre ascendente y descendente", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/canciones?sort=nombreAsc")
		verificar(t, status, respuesta,
			[]string{"BALADA azul", "Balada Roja", "mi_tema", "Otra", "Rock 100%"},
			5, 1, 1)

		status, respuesta = e.pedir(t, yo, "/canciones?sort=nombreDesc")
		verificar(t, status, respuesta,
			[]string{"Rock 100%", "Otra", "mi_tema", "Balada Roja", "BALADA azul"},
			5, 1, 1)
	})

	t.Run("criterios combinados con AND", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, fmt.Sprintf(
			"/canciones?q=balada&proyectoId=%d&sort=nombreAsc", e.proyectoA))
		verificar(t, status, respuesta, []string{"Balada Roja"}, 1, 1, 1)
	})

	t.Run("paginación: página intermedia y última", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/canciones?page=2&pageSize=2")
		verificar(t, status, respuesta, []string{"Rock 100%", "Otra"}, 5, 3, 2)

		status, respuesta = e.pedir(t, yo, "/canciones?page=3&pageSize=2")
		verificar(t, status, respuesta, []string{"mi_tema"}, 5, 3, 3)
	})

	t.Run("paginación sobre el subconjunto filtrado", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/canciones?q=balada&page=2&pageSize=1")
		verificar(t, status, respuesta, []string{"Balada Roja"}, 2, 2, 2)
	})

	t.Run("página fuera de rango", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/canciones?page=9&pageSize=2")
		verificar(t, status, respuesta, []string{}, 5, 3, 9)
	})

	t.Run("usuario sin perfil de integrante", func(t *testing.T) {
		status, _ := e.pedir(t, e.usuarioSinPerfil, "/canciones")
		if status != http.StatusForbidden {
			t.Fatalf("status = %d, se esperaba 403", status)
		}
	})
}
