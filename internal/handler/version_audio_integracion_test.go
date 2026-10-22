package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Tests de integración de HU-VER-B01 (obtener versiones y audio de una
// canción) sobre el escenario de versiones. El 401 sin sesión lo cubre
// TestVersionesYComentarios_RequierenAutenticacion en router_test.go.

// nuevoAjeno crea un usuario con perfil que no integra el proyecto.
func (e *escenarioVersiones) nuevoAjeno(t *testing.T) int64 {
	t.Helper()
	var usuario int64
	sufijo := fmt.Sprintf("%d", time.Now().UnixNano())
	if err := e.db.QueryRow(
		`INSERT INTO usuario (email, idautenticacion) VALUES ($1, gen_random_uuid()) RETURNING codigousuario`,
		"ver-ajeno-"+sufijo+"@test.local",
	).Scan(&usuario); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec(
		`INSERT INTO integrante (codigousuario, nombreintegrante) VALUES ($1, 'Ajeno')`, usuario,
	); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = e.db.Exec(`DELETE FROM integrante WHERE codigousuario = $1`, usuario)
		_, _ = e.db.Exec(`DELETE FROM usuario WHERE codigousuario = $1`, usuario)
	})
	return usuario
}

func (e *escenarioVersiones) rutaAudio(version int64) string {
	return e.ruta(version) + "/audio"
}

func TestVersiones_ListarDevuelveLosDatosDeCadaVersion(t *testing.T) {
	e := prepararEscenarioVersiones(t)

	// Un Músico también puede verlas: alcanza con ser integrante.
	code, _ := e.pedir(t, e.musico, http.MethodGet, e.ruta(0), nil)
	esperarStatus(t, code, nil, http.StatusOK, "")

	versiones := e.listar(t)
	if len(versiones) != 3 {
		t.Fatalf("versiones = %d, quería 3", len(versiones))
	}
	actuales := 0
	for _, v := range versiones {
		if v.EsActual {
			actuales++
			if v.CodigoCancionVersion != e.versiones[2] {
				t.Fatalf("la actual es %d, quería la última (%d)", v.CodigoCancionVersion, e.versiones[2])
			}
		}
		if v.Autor == nil || v.Autor.Nombre != "Ver productor" {
			t.Fatalf("autor de la versión %d = %v", v.NumeroVersion, v.Autor)
		}
		if v.FechaHoraAlta == "" || v.Notas == nil {
			t.Fatalf("versión %d sin fecha o notas: %+v", v.NumeroVersion, v)
		}
	}
	if actuales != 1 {
		t.Fatalf("versiones actuales = %d, quería 1", actuales)
	}
}

func TestVersiones_AudioDevuelveUnaURLFirmada(t *testing.T) {
	e := prepararEscenarioVersiones(t)

	code, body := e.pedir(t, e.musico, http.MethodGet, e.rutaAudio(e.versiones[2]), nil)
	esperarStatus(t, code, body, http.StatusOK, "")

	ruta := fmt.Sprintf("proyectos/%d/v3.mp3", e.proyecto)
	url, _ := body["url"].(string)
	if !strings.HasPrefix(url, "https://storage.test/") || url == ruta {
		t.Fatalf("url = %q: tiene que ser firmada, no la ruta del archivo", url)
	}
	if body["formatoArchivo"] != "mp3" {
		t.Fatalf("formato = %v", body["formatoArchivo"])
	}
	if expira, _ := body["expiraEnSegundos"].(float64); expira <= 0 {
		t.Fatalf("expiraEnSegundos = %v", body["expiraEnSegundos"])
	}
}

func TestVersiones_AudioErrores(t *testing.T) {
	e := prepararEscenarioVersiones(t)
	ajeno := e.nuevoAjeno(t)

	if _, err := e.db.Exec(
		`UPDATE cancionversion SET urlarchivocancionver = NULL WHERE codigocancionversion = $1`,
		e.versiones[0],
	); err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nombre  string
		usuario int64
		ruta    string
		status  int
		mensaje string
	}{
		{"versión sin archivo", e.productor, e.rutaAudio(e.versiones[0]), http.StatusNotFound,
			"La versión no tiene un archivo de audio cargado"},
		{"versión inexistente", e.productor, e.rutaAudio(999999999), http.StatusNotFound, ""},
		{"proyecto inexistente", e.productor,
			fmt.Sprintf("/proyectos/999999999/canciones/%d/versiones/%d/audio", e.cancion, e.versiones[2]),
			http.StatusNotFound, ""},
		{"usuario ajeno al proyecto", ajeno, e.rutaAudio(e.versiones[2]), http.StatusForbidden,
			"No tenés acceso a este proyecto"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			code, body := e.pedir(t, caso.usuario, http.MethodGet, caso.ruta, nil)
			esperarStatus(t, code, body, caso.status, caso.mensaje)
		})
	}
}

func TestVersiones_ListarSinSerIntegranteResponde403(t *testing.T) {
	e := prepararEscenarioVersiones(t)

	code, body := e.pedir(t, e.nuevoAjeno(t), http.MethodGet, e.ruta(0), nil)

	esperarStatus(t, code, body, http.StatusForbidden, "")
}
