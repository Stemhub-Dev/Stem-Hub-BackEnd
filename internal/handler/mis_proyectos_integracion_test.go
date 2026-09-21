package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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

// Tests de integración de GET /proyectos contra PostgreSQL real (ver
// mis_canciones_integracion_test.go: mismas condiciones y mismo
// TEST_DATABASE_URL).

// Códigos de los catálogos que cargan las migraciones 002 y 009.
const (
	estadoSinEmpezar   int64 = 1
	estadoEnDesarrollo int64 = 2
	estadoTerminado    int64 = 3

	tipoSingle int64 = 1
	tipoEP     int64 = 2
	tipoAlbum  int64 = 3

	rolProductor int64 = 1
	rolMusico    int64 = 2
)

// storageDePrueba firma URLs de mentira: el listado solo necesita que la
// portada llegue firmada, no un storage real.
type storageDePrueba struct{}

func (storageDePrueba) Subir(context.Context, string, io.Reader, int64, string) error {
	return nil
}

func (storageDePrueba) ObtenerURLDescarga(_ context.Context, objectKey string, _ time.Duration) (string, error) {
	return "https://storage.test/" + objectKey + "?firma=ok", nil
}

type escenarioMisProyectos struct {
	db *sql.DB

	usuarioConPerfil int64
	usuarioSinPerfil int64

	// Fecha de última modificación esperada de cada proyecto visible.
	fechas map[string]time.Time
}

// prepararEscenarioProyectos carga, con fechas relativas a ahora:
//
//	"Rock Nacional"    Single / Sin Empezar    Productor y propietario; creado hace 10d,
//	                   una canción con versión de hace 1d; logo cargado
//	"Rock Alternativo" EP / En Desarrollo      Músico; creado hace 5d, sin canciones; otro
//	                   integrante se sumó hace 1h (sumar gente no es "modificar")
//	"Jazz 100%"        Album / En Desarrollo   Productor; creado hace 3d; una canción con
//	                   versión de hace 20d y otra dada de baja con versión de hace 1h
//	"Mi_proyecto"      Single / Terminado      Productor; creado hace 2d
//	"Ajeno"            el usuario no participa
//	"Dado de baja"     proyecto dado de baja
//	"Abandonado"       el usuario dejó de participar
//
// Visibles, de la última modificación más reciente a la más vieja:
// Rock Nacional (1d), Mi_proyecto (2d), Jazz 100% (3d), Rock Alternativo (5d).
func prepararEscenarioProyectos(t *testing.T) escenarioMisProyectos {
	t.Helper()

	db := conectarBaseDePrueba(t)
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())
	ahora := time.Now().Truncate(time.Microsecond)
	dia := 24 * time.Hour

	var (
		usuarios    []int64
		integrantes []int64
		proyectos   []int64
		canciones   []int64
		membresias  []int64
	)

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

	nuevoProyecto := func(nombre string, tipo, estado int64, logo *string, dadoDeBaja bool) int64 {
		var baja *time.Time
		if dadoDeBaja {
			baja = &ahora
		}
		var id int64
		debe(db.QueryRow(
			`INSERT INTO proyecto (nombreproyecto, codtipoproy, codestadoproy, logoproyecto, fechahorabajaproyecto)
			 VALUES ($1, $2, $3, $4, $5) RETURNING codigoproyecto`,
			nombre, tipo, estado, logo, baja,
		).Scan(&id))
		proyectos = append(proyectos, id)
		return id
	}

	nuevaMembresia := func(codIntegrante, codProyecto, codRol int64, propietario bool, alta time.Time, dadaDeBaja bool) {
		var baja *time.Time
		if dadaDeBaja {
			baja = &ahora
		}
		var id int64
		debe(db.QueryRow(
			`INSERT INTO integranteproyecto
			   (codintegrante, codigoproyecto, codrol, espropietario, fechahoraaltaintegranteproy, fechahorabajaintegranteproy)
			 VALUES ($1, $2, $3, $4, $5, $6) RETURNING codigointegranteproyecto`,
			codIntegrante, codProyecto, codRol, propietario, alta, baja,
		).Scan(&id))
		membresias = append(membresias, id)
	}

	nuevaCancion := func(codProyecto int64, nombre string, dadaDeBaja bool, versionHace time.Duration) {
		var baja *time.Time
		if dadaDeBaja {
			baja = &ahora
		}
		var id int64
		debe(db.QueryRow(
			`INSERT INTO cancion (codigoproyecto, nombrecancion, fechahorabajacancion)
			 VALUES ($1, $2, $3) RETURNING codigocancion`,
			codProyecto, nombre, baja,
		).Scan(&id))
		canciones = append(canciones, id)

		_, err := db.Exec(
			`INSERT INTO cancionversion (codigocancion, numeroversion, fechahoraaltaversion, formatoarchivocancionver)
			 VALUES ($1, 1, $2, 'mp3')`,
			id, ahora.Add(-versionHace),
		)
		debe(err)
	}

	yo := nuevoUsuario("yo")
	otro := nuevoUsuario("otro")
	sinPerfil := nuevoUsuario("sin-perfil")

	integranteYo := nuevoIntegrante(yo)
	integranteOtro := nuevoIntegrante(otro)

	logo := "proyectos/logo-" + sufijo + ".png"

	rockNacional := nuevoProyecto("Rock Nacional", tipoSingle, estadoSinEmpezar, &logo, false)
	rockAlternativo := nuevoProyecto("Rock Alternativo", tipoEP, estadoEnDesarrollo, nil, false)
	jazz := nuevoProyecto("Jazz 100%", tipoAlbum, estadoEnDesarrollo, nil, false)
	miProyecto := nuevoProyecto("Mi_proyecto", tipoSingle, estadoTerminado, nil, false)
	ajeno := nuevoProyecto("Rock ajeno "+sufijo, tipoSingle, estadoSinEmpezar, nil, false)
	dadoDeBaja := nuevoProyecto("Rock dado de baja "+sufijo, tipoSingle, estadoSinEmpezar, nil, true)
	abandonado := nuevoProyecto("Rock abandonado "+sufijo, tipoSingle, estadoSinEmpezar, nil, false)

	nuevaMembresia(integranteYo, rockNacional, rolProductor, true, ahora.Add(-10*dia), false)
	nuevaMembresia(integranteYo, rockAlternativo, rolMusico, false, ahora.Add(-5*dia), false)
	nuevaMembresia(integranteOtro, rockAlternativo, rolMusico, false, ahora.Add(-time.Hour), false)
	nuevaMembresia(integranteYo, jazz, rolProductor, true, ahora.Add(-3*dia), false)
	nuevaMembresia(integranteYo, miProyecto, rolProductor, true, ahora.Add(-2*dia), false)
	nuevaMembresia(integranteOtro, ajeno, rolProductor, true, ahora.Add(-time.Hour), false)
	nuevaMembresia(integranteYo, dadoDeBaja, rolProductor, true, ahora.Add(-time.Hour), false)
	nuevaMembresia(integranteYo, abandonado, rolProductor, true, ahora.Add(-time.Hour), true)

	nuevaCancion(rockNacional, "Tema", false, 1*dia)
	nuevaCancion(jazz, "Standard", false, 20*dia)
	nuevaCancion(jazz, "Borrada", true, time.Hour)

	return escenarioMisProyectos{
		db:               db,
		usuarioConPerfil: yo,
		usuarioSinPerfil: sinPerfil,
		fechas: map[string]time.Time{
			"Rock Nacional":    ahora.Add(-1 * dia),
			"Rock Alternativo": ahora.Add(-5 * dia),
			"Jazz 100%":        ahora.Add(-3 * dia),
			"Mi_proyecto":      ahora.Add(-2 * dia),
		},
	}
}

func (e escenarioMisProyectos) pedir(
	t *testing.T,
	codigoUsuario int64,
	url string,
) (int, dto.ProyectosPaginadosResponse) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	proyectoHandler := handler.NewProyectoHandler(
		service.NewProyectoService(
			repository.NewProyectoRepository(e.db),
			repository.NewIntegranteRepository(e.db),
			storageDePrueba{},
		),
	)

	router := gin.New()
	router.GET("/proyectos", func(c *gin.Context) {
		c.Set(middleware.UsuarioContextKey, &model.Usuario{CodigoUsuario: codigoUsuario})
		c.Next()
	}, proyectoHandler.Listar)

	grabador := httptest.NewRecorder()
	router.ServeHTTP(grabador, httptest.NewRequest(http.MethodGet, url, nil))

	var cuerpo dto.ProyectosPaginadosResponse
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

func nombresDeProyectos(respuesta dto.ProyectosPaginadosResponse) []string {
	resultado := make([]string, 0, len(respuesta.Data))
	for _, proyecto := range respuesta.Data {
		resultado = append(resultado, proyecto.Nombre)
	}
	return resultado
}

func verificarProyectos(
	t *testing.T,
	status int,
	respuesta dto.ProyectosPaginadosResponse,
	esperados []string,
	totalItems, totalPages, currentPage int,
) {
	t.Helper()

	if status != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", status)
	}
	if obtenidos := nombresDeProyectos(respuesta); !reflect.DeepEqual(obtenidos, esperados) {
		t.Errorf("proyectos = %q, se esperaba %q", obtenidos, esperados)
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

func proyectoPorNombre(t *testing.T, respuesta dto.ProyectosPaginadosResponse, nombre string) dto.ProyectoListadoResponse {
	t.Helper()
	for _, proyecto := range respuesta.Data {
		if proyecto.Nombre == nombre {
			return proyecto
		}
	}
	t.Fatalf("no vino el proyecto %q", nombre)
	return dto.ProyectoListadoResponse{}
}

func TestIntegracionMisProyectos(t *testing.T) {
	e := prepararEscenarioProyectos(t)
	yo := e.usuarioConPerfil
	todos := []string{"Rock Nacional", "Mi_proyecto", "Jazz 100%", "Rock Alternativo"}

	t.Run("sin parámetros: defaults, solo proyectos propios activos, por última modificación", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/proyectos")
		verificarProyectos(t, status, respuesta, todos, 4, 1, 1)
	})

	t.Run("cada proyecto trae tipo, estado, rol, canciones y fecha", func(t *testing.T) {
		_, respuesta := e.pedir(t, yo, "/proyectos")

		rock := proyectoPorNombre(t, respuesta, "Rock Nacional")
		if rock.Tipo != "Single" || rock.Estado != "Sin Empezar" {
			t.Errorf("tipo/estado = %q/%q", rock.Tipo, rock.Estado)
		}
		if rock.NombreRol != "Productor" || !rock.EsPropietario || rock.CantidadCanciones != 1 {
			t.Errorf("rol/propietario/canciones = %q/%v/%d", rock.NombreRol, rock.EsPropietario, rock.CantidadCanciones)
		}

		alternativo := proyectoPorNombre(t, respuesta, "Rock Alternativo")
		if alternativo.NombreRol != "Músico (Artista)" || alternativo.EsPropietario || alternativo.CantidadCanciones != 0 {
			t.Errorf("rol/propietario/canciones = %q/%v/%d",
				alternativo.NombreRol, alternativo.EsPropietario, alternativo.CantidadCanciones)
		}

		// La canción dada de baja no cuenta.
		if jazz := proyectoPorNombre(t, respuesta, "Jazz 100%"); jazz.CantidadCanciones != 1 {
			t.Errorf("canciones de Jazz 100%% = %d, se esperaba 1", jazz.CantidadCanciones)
		}
	})

	t.Run("fecha de última modificación derivada", func(t *testing.T) {
		_, respuesta := e.pedir(t, yo, "/proyectos")

		// Rock Nacional: la versión (hace 1d) es posterior al alta (hace 10d).
		// Jazz 100%: el alta (hace 3d) es posterior a su versión activa (hace
		// 20d); la versión de hace 1h es de una canción dada de baja.
		// Rock Alternativo: el integrante sumado hace 1h no cuenta.
		for nombre, esperada := range e.fechas {
			obtenida := proyectoPorNombre(t, respuesta, nombre).FechaUltimaModificacion
			if !obtenida.Equal(esperada) {
				t.Errorf("fechaUltimaModificacion de %q = %v, se esperaba %v", nombre, obtenida, esperada)
			}
		}
	})

	t.Run("portada firmada solo si el proyecto tiene logo", func(t *testing.T) {
		_, respuesta := e.pedir(t, yo, "/proyectos")

		rock := proyectoPorNombre(t, respuesta, "Rock Nacional")
		if rock.Logo == nil || rock.PortadaURL == nil ||
			*rock.PortadaURL != "https://storage.test/"+*rock.Logo+"?firma=ok" {
			t.Errorf("portada de Rock Nacional = %v (logo %v)", rock.PortadaURL, rock.Logo)
		}
		if jazz := proyectoPorNombre(t, respuesta, "Jazz 100%"); jazz.PortadaURL != nil {
			t.Errorf("un proyecto sin logo no debe tener portada: %v", *jazz.PortadaURL)
		}
	})

	t.Run("búsqueda parcial sin distinguir mayúsculas", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/proyectos?q=rOcK")
		verificarProyectos(t, status, respuesta, []string{"Rock Nacional", "Rock Alternativo"}, 2, 1, 1)
	})

	t.Run("búsqueda con comodines se toma literal", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/proyectos?q=%25")
		verificarProyectos(t, status, respuesta, []string{"Jazz 100%"}, 1, 1, 1)

		status, respuesta = e.pedir(t, yo, "/proyectos?q=_")
		verificarProyectos(t, status, respuesta, []string{"Mi_proyecto"}, 1, 1, 1)
	})

	t.Run("filtro por un estado", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, fmt.Sprintf("/proyectos?estadoId=%d", estadoEnDesarrollo))
		verificarProyectos(t, status, respuesta, []string{"Jazz 100%", "Rock Alternativo"}, 2, 1, 1)
	})

	t.Run("filtro por varios estados", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, fmt.Sprintf(
			"/proyectos?estadoId=%d&estadoId=%d", estadoSinEmpezar, estadoTerminado))
		verificarProyectos(t, status, respuesta, []string{"Rock Nacional", "Mi_proyecto"}, 2, 1, 1)
	})

	t.Run("filtro por tipo", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, fmt.Sprintf("/proyectos?tipoId=%d", tipoSingle))
		verificarProyectos(t, status, respuesta, []string{"Rock Nacional", "Mi_proyecto"}, 2, 1, 1)

		status, respuesta = e.pedir(t, yo, fmt.Sprintf("/proyectos?tipoId=%d&tipoId=%d", tipoEP, tipoAlbum))
		verificarProyectos(t, status, respuesta, []string{"Jazz 100%", "Rock Alternativo"}, 2, 1, 1)
	})

	t.Run("códigos que no existen en el catálogo no devuelven nada", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/proyectos?estadoId=999")
		verificarProyectos(t, status, respuesta, []string{}, 0, 0, 1)
	})

	t.Run("criterios combinados con AND", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, fmt.Sprintf(
			"/proyectos?q=rock&estadoId=%d&tipoId=%d", estadoEnDesarrollo, tipoEP))
		verificarProyectos(t, status, respuesta, []string{"Rock Alternativo"}, 1, 1, 1)

		status, respuesta = e.pedir(t, yo, fmt.Sprintf("/proyectos?q=rock&tipoId=%d", tipoAlbum))
		verificarProyectos(t, status, respuesta, []string{}, 0, 0, 1)
	})

	t.Run("paginación", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/proyectos?page=1&pageSize=3")
		verificarProyectos(t, status, respuesta, []string{"Rock Nacional", "Mi_proyecto", "Jazz 100%"}, 4, 2, 1)

		status, respuesta = e.pedir(t, yo, "/proyectos?page=2&pageSize=3")
		verificarProyectos(t, status, respuesta, []string{"Rock Alternativo"}, 4, 2, 2)
	})

	t.Run("paginación sobre el subconjunto filtrado", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/proyectos?q=rock&page=2&pageSize=1")
		verificarProyectos(t, status, respuesta, []string{"Rock Alternativo"}, 2, 2, 2)
	})

	t.Run("página fuera de rango", func(t *testing.T) {
		status, respuesta := e.pedir(t, yo, "/proyectos?page=5&pageSize=2")
		verificarProyectos(t, status, respuesta, []string{}, 4, 2, 5)
	})

	t.Run("usuario sin perfil de integrante: página vacía", func(t *testing.T) {
		status, respuesta := e.pedir(t, e.usuarioSinPerfil, "/proyectos?page=3")
		verificarProyectos(t, status, respuesta, []string{}, 0, 0, 3)
	})
}
