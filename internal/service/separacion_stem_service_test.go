package service

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/mlservice"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

func (r *stemRepositoryFalso) ListarPorVersion(int64) ([]model.Stem, error) {
	stems := []model.Stem{}
	for _, s := range r.stems {
		stems = append(stems, s)
	}
	return stems, nil
}

type cancionVersionFalsa struct {
	cancionExistenciaFalsa

	version *model.CancionVersion
}

func (c *cancionVersionFalsa) BuscarVersionPorCodigo(int64, int64) (*model.CancionVersion, error) {
	if c.version == nil {
		return nil, sql.ErrNoRows
	}
	return c.version, nil
}

type separacionRepositoryFalso struct {
	repository.SeparacionStemRepository

	enCurso        bool
	yaGenerada     bool
	sinCategorias  bool
	errCompletar   error
	ultima         *model.SeparacionStem
	creada         *model.SeparacionStem
	estados        []string
	mensajeError   string
	stemsGuardados []repository.StemGenerado
}

func (r *separacionRepositoryFalso) Crear(codVersion, codIntegrante int64, cantidad int) (*model.SeparacionStem, error) {
	if r.enCurso {
		return nil, repository.ErrSeparacionEnCurso
	}
	r.creada = &model.SeparacionStem{
		CodSeparacionStem:    5,
		CodigoCancionVersion: codVersion,
		CodIntegrante:        codIntegrante,
		CantidadStems:        cantidad,
		EstadoSeparacion:     model.EstadoSeparacionPendiente,
	}
	return r.creada, nil
}

func (r *separacionRepositoryFalso) MarcarProcesando(int64) error {
	r.estados = append(r.estados, model.EstadoSeparacionProcesando)
	return nil
}

func (r *separacionRepositoryFalso) Completar(_ int64, _ int64, stems []repository.StemGenerado, _ int64) error {
	if r.errCompletar != nil {
		return r.errCompletar
	}
	r.stemsGuardados = stems
	r.estados = append(r.estados, model.EstadoSeparacionCompletada)
	return nil
}

func (r *separacionRepositoryFalso) Fallar(_ int64, mensaje string) error {
	r.mensajeError = mensaje
	r.estados = append(r.estados, model.EstadoSeparacionError)
	return nil
}

func (r *separacionRepositoryFalso) BuscarUltimaPorVersion(int64) (*model.SeparacionStem, error) {
	if r.ultima == nil {
		return nil, sql.ErrNoRows
	}
	return r.ultima, nil
}

func (r *separacionRepositoryFalso) CategoriasPorDefecto() (map[string]int64, error) {
	if r.sinCategorias {
		return map[string]int64{"Voz": 4}, nil
	}
	return map[string]int64{"Batería": 1, "Bajo": 2, "Guitarra": 3, "Voz": 4, "Piano": 5, "Otros": 6}, nil
}

func (r *separacionRepositoryFalso) ExisteStemGeneradoConIA(int64) (bool, error) {
	return r.yaGenerada, nil
}

type mlClienteFalso struct {
	err      error
	recibido *mlservice.SeparacionRequest
}

func (m *mlClienteFalso) SepararStems(_ context.Context, request mlservice.SeparacionRequest) (*mlservice.SeparacionResponse, error) {
	m.recibido = &request
	if m.err != nil {
		return nil, m.err
	}
	respuesta := &mlservice.SeparacionResponse{Status: "completed", ProcessingTimeMs: 1234}
	for nombre, clave := range request.DestinationObjectKeys {
		respuesta.Stems = append(respuesta.Stems, mlservice.StemSeparado{StemName: nombre, ObjectKey: clave, DurationSeconds: 10})
	}
	sort.Slice(respuesta.Stems, func(i, j int) bool { return respuesta.Stems[i].StemName < respuesta.Stems[j].StemName })
	return respuesta, nil
}

func (m *mlClienteFalso) ResumirComentarios(context.Context, mlservice.ResumenRequest) (*mlservice.ResumenResponse, error) {
	return nil, errors.New("no usado")
}

type escenarioSeparacion struct {
	proyecto     *proyectoAccesoFalso
	cancion      *cancionVersionFalsa
	stems        *stemRepositoryFalso
	separaciones *separacionRepositoryFalso
	storage      *storageStemFalso
	ml           *mlClienteFalso
	servicio     SeparacionStemService
}

func nuevoEscenarioSeparacion() *escenarioSeparacion {
	archivo := "proyectos/7/canciones/8/v1.mp3"
	e := &escenarioSeparacion{
		proyecto: &proyectoAccesoFalso{},
		cancion: &cancionVersionFalsa{
			version: &model.CancionVersion{CodigoCancionVersion: 9, URLArchivoCancionVer: &archivo},
		},
		// Un stem "Voz" cargado a mano: el generado tiene que llamarse distinto.
		stems: &stemRepositoryFalso{stems: map[int64]model.Stem{
			10: {CodStem: 10, NombreStem: "Voz", CodCategoriaStem: 4},
		}},
		separaciones: &separacionRepositoryFalso{},
		storage:      &storageStemFalso{},
		ml:           &mlClienteFalso{},
	}
	servicio := NewSeparacionStemService(
		e.separaciones,
		e.stems,
		e.cancion,
		e.proyecto,
		&integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 42}},
		e.storage,
		e.ml,
	).(*separacionStemService)
	servicio.enSegundoPlano = func(f func()) { f() }
	e.servicio = servicio
	return e
}

func cantidad(v int) *int { return &v }

func TestSolicitarSeparacion_CompletaYGuardaStems(t *testing.T) {
	e := nuevoEscenarioSeparacion()

	separacion, err := e.servicio.Solicitar(1, 7, 8, 9, cantidad(4))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if separacion.Estado != model.EstadoSeparacionPendiente || separacion.CantidadStems != 4 {
		t.Fatalf("separación = %+v", separacion)
	}
	if e.proyecto.permisoPide != "GESTIONAR_STEMS" {
		t.Fatalf("permiso pedido = %q", e.proyecto.permisoPide)
	}
	if e.separaciones.creada.CodIntegrante != 42 {
		t.Fatalf("integrante = %d", e.separaciones.creada.CodIntegrante)
	}
	if got := strings.Join(e.separaciones.estados, ","); got != "PROCESANDO,COMPLETADA" {
		t.Fatalf("estados = %s", got)
	}

	recibido := e.ml.recibido
	if recibido.SourceObjectKey != "proyectos/7/canciones/8/v1.mp3" || recibido.StemCount != 4 {
		t.Fatalf("request al ML = %+v", recibido)
	}
	for _, nombre := range []string{"vocals", "drums", "bass", "other"} {
		clave := recibido.DestinationObjectKeys[nombre]
		if !strings.HasPrefix(clave, "proyectos/7/canciones/8/versiones/9/stems/") || !strings.HasSuffix(clave, ".wav") {
			t.Fatalf("destino de %s = %q", nombre, clave)
		}
	}

	nombres := map[string]int64{}
	for _, stem := range e.separaciones.stemsGuardados {
		nombres[stem.Nombre] = stem.CodCategoriaStem
		if stem.Archivo.Formato != "wav" {
			t.Fatalf("formato = %q", stem.Archivo.Formato)
		}
	}
	esperados := map[string]int64{"Voz (IA)": 4, "Batería": 1, "Bajo": 2, "Otros": 6}
	if len(nombres) != len(esperados) {
		t.Fatalf("stems guardados = %v", nombres)
	}
	for nombre, categoria := range esperados {
		if nombres[nombre] != categoria {
			t.Fatalf("stems guardados = %v, se esperaba %s con categoría %d", nombres, nombre, categoria)
		}
	}
}

func TestSolicitarSeparacion_PorDefectoCuatroStems(t *testing.T) {
	e := nuevoEscenarioSeparacion()

	if _, err := e.servicio.Solicitar(1, 7, 8, 9, nil); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if e.ml.recibido.StemCount != 4 {
		t.Fatalf("stem_count = %d", e.ml.recibido.StemCount)
	}
}

func TestSolicitarSeparacion_Rechazos(t *testing.T) {
	casos := []struct {
		nombre   string
		preparar func(*escenarioSeparacion)
		cantidad *int
		esperado error
	}{
		{"cantidad inválida", func(*escenarioSeparacion) {}, cantidad(3), ErrSeparacionCantidadInvalida},
		{"sin permiso", func(e *escenarioSeparacion) { e.proyecto.sinPermiso = true }, nil, ErrStemSinPermiso},
		{"no integrante", func(e *escenarioSeparacion) { e.proyecto.noIntegra = true }, nil, ErrStemSinAcceso},
		{"versión sin archivo", func(e *escenarioSeparacion) { e.cancion.version.URLArchivoCancionVer = nil }, nil, ErrSeparacionVersionSinArchivo},
		{"ya generada", func(e *escenarioSeparacion) { e.separaciones.yaGenerada = true }, nil, ErrSeparacionYaGenerada},
		{"en curso", func(e *escenarioSeparacion) { e.separaciones.enCurso = true }, nil, ErrSeparacionEnCurso},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioSeparacion()
			caso.preparar(e)

			_, err := e.servicio.Solicitar(1, 7, 8, 9, caso.cantidad)

			if !errors.Is(err, caso.esperado) {
				t.Fatalf("err = %v, se esperaba %v", err, caso.esperado)
			}
			if e.ml.recibido != nil {
				t.Fatal("no debería llamar al servicio de IA")
			}
		})
	}
}

func TestSolicitarSeparacion_FallaElServicioDeIA(t *testing.T) {
	e := nuevoEscenarioSeparacion()
	e.ml.err = &mlservice.ErrorServicioML{Status: 504, Codigo: mlservice.CodigoSeparacionTimeout}

	if _, err := e.servicio.Solicitar(1, 7, 8, 9, cantidad(2)); err != nil {
		t.Fatalf("el pedido no debería fallar, la separación sí: %v", err)
	}

	if got := strings.Join(e.separaciones.estados, ","); got != "PROCESANDO,ERROR" {
		t.Fatalf("estados = %s", got)
	}
	if e.separaciones.mensajeError != mensajeSeparacionTimeout {
		t.Fatalf("mensaje = %q", e.separaciones.mensajeError)
	}
}

func TestSolicitarSeparacion_SiNoSeGuardaBorraLosArchivos(t *testing.T) {
	e := nuevoEscenarioSeparacion()
	e.separaciones.sinCategorias = true

	if _, err := e.servicio.Solicitar(1, 7, 8, 9, cantidad(2)); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if e.separaciones.mensajeError != mensajeSeparacionSinCategorias {
		t.Fatalf("mensaje = %q", e.separaciones.mensajeError)
	}
	if len(e.storage.eliminados) != 2 {
		t.Fatalf("eliminados = %v", e.storage.eliminados)
	}
}

func TestObtenerUltimaSeparacion_SinSeparaciones(t *testing.T) {
	e := nuevoEscenarioSeparacion()

	_, err := e.servicio.ObtenerUltima(1, 7, 8, 9)

	if !errors.Is(err, ErrSeparacionNoEncontrada) {
		t.Fatalf("err = %v", err)
	}
	if e.proyecto.permisoPide != "" {
		t.Fatalf("consultar no debería pedir permiso, pidió %q", e.proyecto.permisoPide)
	}
}

func TestNombreStemLibre(t *testing.T) {
	usados := map[string]bool{"voz": true, "voz (ia)": true}

	if got := nombreStemLibre("Voz", usados); got != "Voz (IA 2)" {
		t.Fatalf("nombre = %q", got)
	}
	if got := nombreStemLibre("Bajo", usados); got != "Bajo" {
		t.Fatalf("nombre = %q", got)
	}
}
