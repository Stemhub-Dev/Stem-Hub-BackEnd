package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
)

type preguntaFrecuenteRepositoryFalso struct {
	preguntas []model.PreguntaFrecuente
	err       error
}

func (r *preguntaFrecuenteRepositoryFalso) ListarActivas() ([]model.PreguntaFrecuente, error) {
	return r.preguntas, r.err
}

func TestListarPreguntasFrecuentes_MapeaLosCampos(t *testing.T) {
	servicio := NewAyudaService(&preguntaFrecuenteRepositoryFalso{
		preguntas: []model.PreguntaFrecuente{{
			CodigoPreguntaFrecuente: 3,
			FuncionalidadPrincipal:  "Registro y acceso",
			PreguntaFrecuente:       "¿Cómo creo una cuenta?",
			RespuestaFrecuente:      "Con tu email.",
		}},
	}, "")

	preguntas, err := servicio.ListarPreguntasFrecuentes()

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(preguntas) != 1 {
		t.Fatalf("len = %d", len(preguntas))
	}
	pregunta := preguntas[0]
	if pregunta.ID != 3 || pregunta.FuncionalidadPrincipal != "Registro y acceso" ||
		pregunta.PreguntaFrecuente != "¿Cómo creo una cuenta?" ||
		pregunta.RespuestaPreguntaFrecuente != "Con tu email." {
		t.Fatalf("mapeo incorrecto: %+v", pregunta)
	}
}

func TestListarPreguntasFrecuentes_SinPreguntasDevuelveListaVacia(t *testing.T) {
	preguntas, err := NewAyudaService(&preguntaFrecuenteRepositoryFalso{}, "").ListarPreguntasFrecuentes()

	if err != nil || preguntas == nil || len(preguntas) != 0 {
		t.Fatalf("preguntas/err = %#v/%v, se esperaba lista vacía no nil", preguntas, err)
	}
}

func TestListarPreguntasFrecuentes_PropagaError(t *testing.T) {
	errBase := errors.New("fallo de base")

	_, err := NewAyudaService(&preguntaFrecuenteRepositoryFalso{err: errBase}, "").ListarPreguntasFrecuentes()

	if !errors.Is(err, errBase) {
		t.Fatalf("err = %v, se esperaba %v", err, errBase)
	}
}

func TestObtenerRutaManual_DevuelveLaRutaSiExiste(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "manual.pdf")
	if err := os.WriteFile(ruta, []byte("%PDF"), 0o600); err != nil {
		t.Fatal(err)
	}

	obtenida, err := NewAyudaService(&preguntaFrecuenteRepositoryFalso{}, ruta).ObtenerRutaManual()

	if err != nil || obtenida != ruta {
		t.Fatalf("ruta/err = %q/%v", obtenida, err)
	}
}

func TestObtenerRutaManual_SinArchivoNoEstaDisponible(t *testing.T) {
	directorio := t.TempDir()

	casos := map[string]string{
		"inexistente": filepath.Join(directorio, "no-existe.pdf"),
		"directorio":  directorio,
		"sin ruta":    "",
	}

	for nombre, ruta := range casos {
		t.Run(nombre, func(t *testing.T) {
			_, err := NewAyudaService(&preguntaFrecuenteRepositoryFalso{}, ruta).ObtenerRutaManual()

			if !errors.Is(err, ErrManualUsuarioNoDisponible) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
