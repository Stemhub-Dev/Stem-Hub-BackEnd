package service

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// Falsos para StemService: embeben la interfaz real (nil) e implementan solo
// lo que usa el service.

type proyectoAccesoFalso struct {
	repository.ProyectoRepository

	noExiste    bool
	noIntegra   bool
	sinPermiso  bool
	permisoPide string
}

func (p *proyectoAccesoFalso) ExisteProyectoActivo(int64) (bool, error) {
	return !p.noExiste, nil
}

func (p *proyectoAccesoFalso) EsIntegranteActivo(int64, int64) (bool, error) {
	return !p.noIntegra, nil
}

func (p *proyectoAccesoFalso) PuedeRealizarEnProyecto(_, _ int64, permiso string) (bool, error) {
	p.permisoPide = permiso
	return !p.sinPermiso, nil
}

type cancionExistenciaFalsa struct {
	repository.CancionRepository

	versionNoExiste bool
}

func (c *cancionExistenciaFalsa) ExisteCancionActivaEnProyecto(int64, int64) (bool, error) {
	return true, nil
}

func (c *cancionExistenciaFalsa) ExisteVersionActivaEnCancion(int64, int64) (bool, error) {
	return !c.versionNoExiste, nil
}

type stemRepositoryFalso struct {
	repository.StemRepository

	stems map[int64]model.Stem
	// Versiones con las que se buscó cada stem.
	versionesBuscadas []int64
	categorias        []model.CategoriaStem
	errCrear          error
	errActualizar     error

	creado       *repository.ArchivoStemGuardado
	nuevaCatPide *string
	actualizado  *repository.ArchivoStemGuardado
	cambios      *repository.CambiosStem
	eliminado    int64
}

func (r *stemRepositoryFalso) BuscarCategoria(_ int64, cod int64) (*model.CategoriaStem, error) {
	for _, c := range r.categorias {
		if c.CodCategoriaStem == cod {
			return &c, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (r *stemRepositoryFalso) ExisteCategoriaConNombre(_ int64, nombre string) (bool, error) {
	for _, c := range r.categorias {
		if strings.EqualFold(c.NombreCategoriaStem, strings.TrimSpace(nombre)) {
			return true, nil
		}
	}
	return false, nil
}

func (r *stemRepositoryFalso) ExisteNombreEnVersion(_ int64, nombre string, excluido int64) (bool, error) {
	for cod, s := range r.stems {
		if cod != excluido && strings.EqualFold(s.NombreStem, nombre) {
			return true, nil
		}
	}
	return false, nil
}

// Como el repositorio real, solo encuentra el stem dentro de su versión.
func (r *stemRepositoryFalso) BuscarPorCodigo(version int64, cod int64) (*model.Stem, error) {
	r.versionesBuscadas = append(r.versionesBuscadas, version)
	s, ok := r.stems[cod]
	if !ok || s.CodigoCancionVersion != version {
		return nil, sql.ErrNoRows
	}
	return &s, nil
}

func (r *stemRepositoryFalso) Crear(
	_ int64,
	_ int64,
	codigoCancionVersion int64,
	nombre string,
	codCategoria int64,
	nuevaCategoria *string,
	archivo repository.ArchivoStemGuardado,
) (int64, error) {
	if r.errCrear != nil {
		return 0, r.errCrear
	}
	r.creado = &archivo
	r.nuevaCatPide = nuevaCategoria
	r.stems[99] = model.Stem{
		CodStem:              99,
		CodigoCancionVersion: codigoCancionVersion,
		NombreStem:           nombre,
		CodCategoriaStem:     codCategoria,
		URLArchivoStem:       archivo.URL,
		FormatoArchivoStem:   archivo.Formato,
		NombreArchivoStem:    archivo.NombreOriginal,
	}
	return 99, nil
}

func (r *stemRepositoryFalso) Actualizar(_ int64, cod int64, cambios repository.CambiosStem) error {
	if r.errActualizar != nil {
		return r.errActualizar
	}
	r.cambios = &cambios
	r.actualizado = cambios.Archivo
	s := r.stems[cod]
	s.NombreStem = cambios.Nombre
	s.NombreArchivoStem = cambios.NombreArchivo
	s.CodCategoriaStem = cambios.CodCategoriaStem
	if cambios.Archivo != nil {
		s.URLArchivoStem = cambios.Archivo.URL
		s.FormatoArchivoStem = cambios.Archivo.Formato
		s.GeneradoConIA = false
	}
	r.stems[cod] = s
	return nil
}

func (r *stemRepositoryFalso) Eliminar(cod int64) error {
	r.eliminado = cod
	delete(r.stems, cod)
	return nil
}

type storageStemFalso struct {
	errSubir   error
	subidos    []string
	eliminados []string
}

func (s *storageStemFalso) Subir(_ context.Context, key string, _ io.Reader, _ int64, _ string) error {
	if s.errSubir != nil {
		return s.errSubir
	}
	s.subidos = append(s.subidos, key)
	return nil
}

func (s *storageStemFalso) ObtenerURLDescarga(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://storage.test/" + key, nil
}

func (s *storageStemFalso) Eliminar(_ context.Context, key string) error {
	s.eliminados = append(s.eliminados, key)
	return nil
}

type escenarioStems struct {
	proyecto *proyectoAccesoFalso
	cancion  *cancionExistenciaFalsa
	stems    *stemRepositoryFalso
	storage  *storageStemFalso
	servicio StemService
}

func nuevoEscenarioStems() *escenarioStems {
	e := &escenarioStems{
		proyecto: &proyectoAccesoFalso{},
		cancion:  &cancionExistenciaFalsa{},
		stems: &stemRepositoryFalso{
			stems: map[int64]model.Stem{
				10: {
					CodStem:              10,
					CodigoCancionVersion: 9,
					NombreStem:           "Bajo",
					CodCategoriaStem:     2,
					URLArchivoStem:       "viejo.wav",
					FormatoArchivoStem:   "wav",
					NombreArchivoStem:    "bass.wav",
					GeneradoConIA:        true,
				},
				// De otra versión de la misma canción.
				11: {CodStem: 11, CodigoCancionVersion: 77, NombreStem: "Voz", FormatoArchivoStem: "wav"},
			},
			categorias: []model.CategoriaStem{
				{CodCategoriaStem: 2, NombreCategoriaStem: "Bajo"},
				{CodCategoriaStem: 3, NombreCategoriaStem: "Guitarra"},
			},
		},
		storage: &storageStemFalso{},
	}
	e.servicio = NewStemService(
		e.stems,
		e.cancion,
		e.proyecto,
		&integranteRepositoryFalso{integrante: &model.Integrante{CodIntegrante: 42}},
		e.storage,
	)
	return e
}

func archivoDePrueba(nombre string, tamano int64) ArchivoAudio {
	return ArchivoAudio{Contenido: strings.NewReader("audio"), NombreOriginal: nombre, Tamano: tamano}
}

func codigo(v int64) *int64 { return &v }

func texto(v string) *string { return &v }

func TestCrearStem_ConCategoriaExistente(t *testing.T) {
	e := nuevoEscenarioStems()

	stem, err := e.servicio.Crear(1, 7, 8, 9,
		CrearStemRequest{Nombre: "  Guitarra 2 ", CodCategoriaStem: codigo(3)},
		archivoDePrueba("gtr.wav", 100),
	)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if stem.Nombre != "Guitarra 2" || stem.GeneradoConIA {
		t.Fatalf("stem = %+v", stem)
	}
	if e.proyecto.permisoPide != "GESTIONAR_STEMS" {
		t.Fatalf("permiso pedido = %q", e.proyecto.permisoPide)
	}
	if len(e.storage.subidos) != 1 ||
		!strings.HasPrefix(e.storage.subidos[0], "proyectos/7/canciones/8/versiones/9/stems/") ||
		!strings.HasSuffix(e.storage.subidos[0], ".wav") {
		t.Fatalf("subidos = %v", e.storage.subidos)
	}
	if e.stems.creado.NombreOriginal != "gtr.wav" || e.stems.nuevaCatPide != nil {
		t.Fatalf("creado = %+v, nueva categoría = %v", e.stems.creado, e.stems.nuevaCatPide)
	}
}

func TestCrearStem_ConCategoriaNueva(t *testing.T) {
	e := nuevoEscenarioStems()

	_, err := e.servicio.Crear(1, 7, 8, 9,
		CrearStemRequest{Nombre: "Teclados", NuevaCategoria: texto(" Teclados ")},
		archivoDePrueba("keys.flac", 100),
	)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if e.stems.nuevaCatPide == nil || *e.stems.nuevaCatPide != "Teclados" {
		t.Fatalf("nueva categoría = %v", e.stems.nuevaCatPide)
	}
}

func TestCrearStem_Validaciones(t *testing.T) {
	ConfigurarTamanosMaximos(100, 1)
	t.Cleanup(func() { ConfigurarTamanosMaximos(100, 100) })

	casos := []struct {
		nombre   string
		request  CrearStemRequest
		archivo  ArchivoAudio
		ajustar  func(*escenarioStems)
		esperado error
	}{
		{"sin nombre", CrearStemRequest{Nombre: " ", CodCategoriaStem: codigo(3)}, archivoDePrueba("a.wav", 1), nil, ErrStemNombreObligatorio},
		{"sin categoría", CrearStemRequest{Nombre: "X"}, archivoDePrueba("a.wav", 1), nil, ErrStemCategoriaObligatoria},
		{"categoría nueva vacía", CrearStemRequest{Nombre: "X", NuevaCategoria: texto("  ")}, archivoDePrueba("a.wav", 1), nil, ErrStemCategoriaObligatoria},
		{"sin archivo", CrearStemRequest{Nombre: "X", CodCategoriaStem: codigo(3)}, ArchivoAudio{}, nil, ErrStemArchivoObligatorio},
		{"archivo vacío", CrearStemRequest{Nombre: "X", CodCategoriaStem: codigo(3)}, archivoDePrueba("a.wav", 0), nil, ErrStemArchivoVacio},
		{"formato", CrearStemRequest{Nombre: "X", CodCategoriaStem: codigo(3)}, archivoDePrueba("a.aiff", 1), nil, ErrStemFormatoInvalido},
		{"supera el límite configurado", CrearStemRequest{Nombre: "X", CodCategoriaStem: codigo(3)}, archivoDePrueba("a.wav", 2*1024*1024), nil, ErrStemArchivoDemasiadoGrande},
		{"categoría de otro proyecto", CrearStemRequest{Nombre: "X", CodCategoriaStem: codigo(77)}, archivoDePrueba("a.wav", 1), nil, ErrStemCategoriaNoEncontrada},
		{"categoría repetida", CrearStemRequest{Nombre: "X", NuevaCategoria: texto("bajo")}, archivoDePrueba("a.wav", 1), nil, ErrStemCategoriaDuplicada},
		{"nombre repetido", CrearStemRequest{Nombre: "BAJO", CodCategoriaStem: codigo(2)}, archivoDePrueba("a.wav", 1), nil, ErrStemNombreDuplicado},
		{"sin permiso", CrearStemRequest{Nombre: "X", CodCategoriaStem: codigo(3)}, archivoDePrueba("a.wav", 1), func(e *escenarioStems) { e.proyecto.sinPermiso = true }, ErrStemSinPermiso},
		{"no integra el proyecto", CrearStemRequest{Nombre: "X", CodCategoriaStem: codigo(3)}, archivoDePrueba("a.wav", 1), func(e *escenarioStems) { e.proyecto.noIntegra = true }, ErrStemSinAcceso},
		{"versión inexistente", CrearStemRequest{Nombre: "X", CodCategoriaStem: codigo(3)}, archivoDePrueba("a.wav", 1), func(e *escenarioStems) { e.cancion.versionNoExiste = true }, ErrStemVersionNoEncontrada},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioStems()
			if caso.ajustar != nil {
				caso.ajustar(e)
			}

			_, err := e.servicio.Crear(1, 7, 8, 9, caso.request, caso.archivo)

			if !errors.Is(err, caso.esperado) {
				t.Fatalf("err = %v, se esperaba %v", err, caso.esperado)
			}
			if len(e.storage.subidos) != 0 {
				t.Fatalf("no debería subir nada, subió %v", e.storage.subidos)
			}
		})
	}
}

func TestCrearStem_SiFallaLaBaseBorraElArchivoSubido(t *testing.T) {
	e := nuevoEscenarioStems()
	e.stems.errCrear = errors.New("db caída")

	_, err := e.servicio.Crear(1, 7, 8, 9,
		CrearStemRequest{Nombre: "X", CodCategoriaStem: codigo(3)},
		archivoDePrueba("a.wav", 1),
	)

	if err == nil {
		t.Fatal("se esperaba error")
	}
	if len(e.storage.eliminados) != 1 || e.storage.eliminados[0] != e.storage.subidos[0] {
		t.Fatalf("subidos = %v, eliminados = %v", e.storage.subidos, e.storage.eliminados)
	}
}

func TestEditarStem_SoloNombreConservaArchivoEIA(t *testing.T) {
	e := nuevoEscenarioStems()

	stem, err := e.servicio.Editar(1, 7, 8, 9, 10, EditarStemRequest{Nombre: "Bajo eléctrico"}, nil)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if stem.Nombre != "Bajo eléctrico" || !stem.GeneradoConIA ||
		stem.NombreArchivo != "bass.wav" || stem.CodCategoriaStem != 2 {
		t.Fatalf("stem = %+v", stem)
	}
	if e.stems.actualizado != nil || len(e.storage.subidos) != 0 || len(e.storage.eliminados) != 0 {
		t.Fatal("no debería tocar el archivo")
	}
}

func TestEditarStem_ReemplazoBorraElArchivoViejoYQuitaIA(t *testing.T) {
	e := nuevoEscenarioStems()
	archivo := archivoDePrueba("bajo_v2.mp3", 10)

	stem, err := e.servicio.Editar(1, 7, 8, 9, 10, EditarStemRequest{Nombre: "Bajo"}, &archivo)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if stem.GeneradoConIA || stem.NombreArchivo != "bajo_v2.mp3" {
		t.Fatalf("stem = %+v", stem)
	}
	if len(e.storage.eliminados) != 1 || e.storage.eliminados[0] != "viejo.wav" {
		t.Fatalf("eliminados = %v", e.storage.eliminados)
	}
}

func TestEditarStem_RenombraElArchivoConservandoLaExtension(t *testing.T) {
	casos := []struct {
		nombre        string
		nombreArchivo string
		archivo       *ArchivoAudio
		esperado      string
	}{
		{"sin extensión", " bajo final ", nil, "bajo final.wav"},
		{"con la extensión ya escrita", "bajo final.WAV", nil, "bajo final.wav"},
		{"con otra extensión", "bajo.mp3", nil, "bajo.mp3.wav"},
		{"junto con un archivo nuevo", "bajo final", func() *ArchivoAudio { a := archivoDePrueba("x.flac", 10); return &a }(), "bajo final.flac"},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioStems()

			stem, err := e.servicio.Editar(1, 7, 8, 9, 10,
				EditarStemRequest{Nombre: "Bajo", NombreArchivo: texto(caso.nombreArchivo)},
				caso.archivo,
			)

			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if stem.NombreArchivo != caso.esperado {
				t.Fatalf("nombre de archivo = %q, se esperaba %q", stem.NombreArchivo, caso.esperado)
			}
			if caso.archivo == nil && !stem.GeneradoConIA {
				t.Fatal("renombrar el archivo no debería quitar la marca de IA")
			}
		})
	}
}

func TestEditarStem_CambiaACategoriaExistenteConservandoIA(t *testing.T) {
	e := nuevoEscenarioStems()

	stem, err := e.servicio.Editar(1, 7, 8, 9, 10,
		EditarStemRequest{Nombre: "Bajo", CodCategoriaStem: codigo(3)},
		nil,
	)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if stem.CodCategoriaStem != 3 || !stem.GeneradoConIA {
		t.Fatalf("stem = %+v", stem)
	}
	if e.stems.cambios.NuevaCategoria != nil {
		t.Fatalf("no debería crear categoría: %v", e.stems.cambios.NuevaCategoria)
	}
}

func TestEditarStem_CambiaACategoriaNueva(t *testing.T) {
	e := nuevoEscenarioStems()

	_, err := e.servicio.Editar(1, 7, 8, 9, 10,
		EditarStemRequest{Nombre: "Bajo", NuevaCategoria: texto(" Sintes ")},
		nil,
	)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if e.stems.cambios.NuevaCategoria == nil || *e.stems.cambios.NuevaCategoria != "Sintes" {
		t.Fatalf("nueva categoría = %v", e.stems.cambios.NuevaCategoria)
	}
}

func TestEditarStem_MismaCategoriaNoLaVuelveAValidar(t *testing.T) {
	e := nuevoEscenarioStems()
	// La categoría actual del stem (2) ya no figura entre las utilizables.
	e.stems.categorias = e.stems.categorias[1:]

	stem, err := e.servicio.Editar(1, 7, 8, 9, 10,
		EditarStemRequest{Nombre: "Bajo", CodCategoriaStem: codigo(2)},
		nil,
	)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if stem.CodCategoriaStem != 2 {
		t.Fatalf("stem = %+v", stem)
	}
}

func TestEditarStem_Validaciones(t *testing.T) {
	casos := []struct {
		nombre   string
		request  EditarStemRequest
		codStem  int64
		ajustar  func(*escenarioStems)
		esperado error
	}{
		{"stem inexistente", EditarStemRequest{Nombre: "X"}, 404, nil, ErrStemNoEncontrado},
		{"sin nombre", EditarStemRequest{Nombre: " "}, 10, nil, ErrStemNombreObligatorio},
		{"nombre de archivo vacío", EditarStemRequest{Nombre: "X", NombreArchivo: texto(" .wav ")}, 10, nil, ErrStemNombreArchivoObligatorio},
		{"nombre de archivo con barras", EditarStemRequest{Nombre: "X", NombreArchivo: texto("a/b")}, 10, nil, ErrStemNombreArchivoInvalido},
		{"nombre de archivo largo", EditarStemRequest{Nombre: "X", NombreArchivo: texto(strings.Repeat("a", 252))}, 10, nil, ErrStemNombreArchivoLargo},
		{"categoría de otro proyecto", EditarStemRequest{Nombre: "X", CodCategoriaStem: codigo(77)}, 10, nil, ErrStemCategoriaNoEncontrada},
		{"categoría nueva vacía", EditarStemRequest{Nombre: "X", NuevaCategoria: texto("  ")}, 10, nil, ErrStemCategoriaObligatoria},
		{"categoría nueva repetida", EditarStemRequest{Nombre: "X", NuevaCategoria: texto("guitarra")}, 10, nil, ErrStemCategoriaDuplicada},
		{"sin permiso", EditarStemRequest{Nombre: "X", CodCategoriaStem: codigo(3)}, 10, func(e *escenarioStems) { e.proyecto.sinPermiso = true }, ErrStemSinPermiso},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioStems()
			if caso.ajustar != nil {
				caso.ajustar(e)
			}

			_, err := e.servicio.Editar(1, 7, 8, 9, caso.codStem, caso.request, nil)

			if !errors.Is(err, caso.esperado) {
				t.Fatalf("err = %v, se esperaba %v", err, caso.esperado)
			}
			if e.stems.cambios != nil {
				t.Fatal("no debería actualizar el stem")
			}
		})
	}
}

func TestEliminarStem_BorraFilaYArchivo(t *testing.T) {
	e := nuevoEscenarioStems()

	if err := e.servicio.Eliminar(1, 7, 8, 9, 10); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if e.stems.eliminado != 10 {
		t.Fatalf("eliminado = %d", e.stems.eliminado)
	}
	if len(e.storage.eliminados) != 1 || e.storage.eliminados[0] != "viejo.wav" {
		t.Fatalf("eliminados = %v", e.storage.eliminados)
	}
}

func TestEliminarStem_SinPermiso(t *testing.T) {
	e := nuevoEscenarioStems()
	e.proyecto.sinPermiso = true

	if err := e.servicio.Eliminar(1, 7, 8, 9, 10); !errors.Is(err, ErrStemSinPermiso) {
		t.Fatalf("err = %v", err)
	}
	if e.stems.eliminado != 0 {
		t.Fatal("no debería eliminar")
	}
}

func TestObtenerURLAudioStem_NoPideGestionar(t *testing.T) {
	e := nuevoEscenarioStems()
	e.proyecto.sinPermiso = true

	audio, err := e.servicio.ObtenerURLAudio(1, 7, 8, 9, 10)

	if err != nil {
		t.Fatalf("un integrante sin GESTIONAR_STEMS debería poder escuchar: %v", err)
	}
	if audio.URL != "https://storage.test/viejo.wav" {
		t.Fatalf("url = %q", audio.URL)
	}
}

// HU-ABM-04-04: solo un integrante del proyecto obtiene la URL del audio, y
// solo de un stem de esa versión.
func TestObtenerURLAudioStem_Restricciones(t *testing.T) {
	t.Run("no integrante", func(t *testing.T) {
		e := nuevoEscenarioStems()
		e.proyecto.noIntegra = true

		if _, err := e.servicio.ObtenerURLAudio(1, 7, 8, 9, 10); !errors.Is(err, ErrStemSinAcceso) {
			t.Fatalf("err = %v, quería ErrStemSinAcceso", err)
		}
	})

	t.Run("versión de otra canción", func(t *testing.T) {
		e := nuevoEscenarioStems()
		e.cancion.versionNoExiste = true

		if _, err := e.servicio.ObtenerURLAudio(1, 7, 8, 9, 10); !errors.Is(err, ErrStemVersionNoEncontrada) {
			t.Fatalf("err = %v, quería ErrStemVersionNoEncontrada", err)
		}
	})

	t.Run("stem que no es de la versión", func(t *testing.T) {
		e := nuevoEscenarioStems()

		// El stem 11 existe, pero es de la versión 77.
		if _, err := e.servicio.ObtenerURLAudio(1, 7, 8, 9, 11); !errors.Is(err, ErrStemNoEncontrado) {
			t.Fatalf("err = %v, quería ErrStemNoEncontrado", err)
		}
		if len(e.stems.versionesBuscadas) == 0 || e.stems.versionesBuscadas[0] != 9 {
			t.Fatalf("versión buscada = %v, quería 9", e.stems.versionesBuscadas)
		}
	})
}

// Otra alta en paralelo ganó la carrera: el índice único la frena y se
// responde el mismo duplicado que el chequeo previo (409), sin dejar el
// archivo subido huérfano.
func TestCrearStem_DuplicadoDetectadoPorLaBaseEsNombreDuplicado(t *testing.T) {
	casos := map[error]error{
		repository.ErrNombreStemDuplicado:    ErrStemNombreDuplicado,
		repository.ErrCategoriaStemDuplicada: ErrStemCategoriaDuplicada,
	}

	for errRepo, quiero := range casos {
		t.Run(quiero.Error(), func(t *testing.T) {
			e := nuevoEscenarioStems()
			e.stems.errCrear = errRepo

			_, err := e.servicio.Crear(1, 7, 8, 9,
				CrearStemRequest{Nombre: "Contrabajo", CodCategoriaStem: codigo(3)},
				archivoDePrueba("contrabajo.wav", 100),
			)

			if !errors.Is(err, quiero) {
				t.Fatalf("err = %v, quería %v", err, quiero)
			}
			if len(e.storage.eliminados) != 1 || e.storage.eliminados[0] != e.storage.subidos[0] {
				t.Fatalf("eliminados = %v, subidos = %v", e.storage.eliminados, e.storage.subidos)
			}
		})
	}
}

// Lo mismo al editar: otro renombre en paralelo ganó la carrera. El archivo
// de reemplazo que se subió se borra.
func TestEditarStem_DuplicadoDetectadoPorLaBaseEsNombreDuplicado(t *testing.T) {
	e := nuevoEscenarioStems()
	e.stems.errActualizar = repository.ErrNombreStemDuplicado
	archivo := archivoDePrueba("nuevo.wav", 100)

	_, err := e.servicio.Editar(1, 7, 8, 9, 10, EditarStemRequest{Nombre: "Contrabajo"}, &archivo)

	if !errors.Is(err, ErrStemNombreDuplicado) {
		t.Fatalf("err = %v, quería ErrStemNombreDuplicado", err)
	}
	if len(e.storage.subidos) != 1 || len(e.storage.eliminados) != 1 || e.storage.eliminados[0] != e.storage.subidos[0] {
		t.Fatalf("subidos = %v, eliminados = %v", e.storage.subidos, e.storage.eliminados)
	}
}
