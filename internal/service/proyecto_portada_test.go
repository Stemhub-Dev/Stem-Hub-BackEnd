package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
)

// portadaRepositoryFalso cubre lo que usan el alta y la modificación de un
// proyecto. Crear simula la transacción: si subirLogo falla, no queda nada
// creado.
type portadaRepositoryFalso struct {
	repository.ProyectoRepository

	logoActual  *string
	propietario bool

	crearLlamado     bool
	creado           bool
	logoCreado       *string
	actualizado      bool
	logoActualizado  *string
	codigoProyectoID int64
}

func (r *portadaRepositoryFalso) ExisteTipoProyectoActivo(int64) (bool, error) {
	return true, nil
}

func (r *portadaRepositoryFalso) ObtenerAmbitoRolActivo(int64) (string, error) {
	return ambitoRolProyecto, nil
}

func (r *portadaRepositoryFalso) ExistenGeneros([]int64) (bool, error) {
	return true, nil
}

func (r *portadaRepositoryFalso) Crear(
	_ int64,
	_ string,
	_ *string,
	_ int64,
	_ int64,
	_ []int64,
	_ int64,
	_ string,
	subirLogo func(codigoProyecto int64) (string, error),
) (int64, error) {

	r.crearLlamado = true

	if subirLogo != nil {
		clave, err := subirLogo(r.codigoProyectoID)
		if err != nil {
			return 0, err
		}
		r.logoCreado = &clave
	}

	r.creado = true

	return r.codigoProyectoID, nil
}

func (r *portadaRepositoryFalso) ExisteProyectoActivo(int64) (bool, error) {
	return true, nil
}

func (r *portadaRepositoryFalso) EsPropietarioActivo(int64, int64) (bool, error) {
	return r.propietario, nil
}

func (r *portadaRepositoryFalso) ObtenerDetalle(codigo int64) (*model.Proyecto, string, string, error) {
	return &model.Proyecto{
		CodigoProyecto: codigo,
		NombreProyecto: "Rock Nacional",
		LogoProyecto:   r.logoActual,
		CodEstadoProy:  1,
		CodTipoProy:    1,
	}, "Single", "Sin Empezar", nil
}

func (r *portadaRepositoryFalso) ListarGenerosProyecto(int64) ([]dto.GeneroProyectoResponse, error) {
	return []dto.GeneroProyectoResponse{{CodigoGenero: 3}}, nil
}

func (r *portadaRepositoryFalso) Actualizar(
	_ int64,
	_ string,
	_ *string,
	logoObjectKey *string,
	_ int64,
	_ int64,
	_ []int64,
) error {
	r.actualizado = true
	r.logoActualizado = logoObjectKey
	return nil
}

func nuevoServicioPortada(logoActual *string) (*portadaRepositoryFalso, *storageStemFalso, ProyectoService) {
	repo := &portadaRepositoryFalso{
		logoActual:       logoActual,
		propietario:      true,
		codigoProyectoID: 9,
	}
	almacenamiento := &storageStemFalso{}
	return repo, almacenamiento, NewProyectoService(repo, integranteActivo(), almacenamiento)
}

func imagen(nombre string, tamano int64) *ArchivoImagen {
	return &ArchivoImagen{
		Contenido:      strings.NewReader("imagen"),
		NombreOriginal: nombre,
		Tamano:         tamano,
	}
}

func pedidoAltaProyecto() dto.CrearProyectoRequest {
	return dto.CrearProyectoRequest{
		Nombre:             "Rock Nacional",
		CodigoTipoProyecto: 1,
		CodigosGeneros:     []int64{3},
		CodRol:             2,
	}
}

func TestCrearProyecto_ConPortadaLaSubeConLaClaveDelProyecto(t *testing.T) {
	repo, almacenamiento, servicio := nuevoServicioPortada(nil)

	_, err := servicio.CrearProyecto(1, pedidoAltaProyecto(), imagen("tapa.PNG", 1024))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(almacenamiento.subidos) != 1 || almacenamiento.subidos[0] != "proyectos/9/logo.png" {
		t.Fatalf("subidos = %v", almacenamiento.subidos)
	}
	if repo.logoCreado == nil || *repo.logoCreado != "proyectos/9/logo.png" {
		t.Fatalf("logo guardado = %v", repo.logoCreado)
	}
}

func TestCrearProyecto_SinPortadaNoSubeNada(t *testing.T) {
	repo, almacenamiento, servicio := nuevoServicioPortada(nil)

	if _, err := servicio.CrearProyecto(1, pedidoAltaProyecto(), nil); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !repo.creado || repo.logoCreado != nil || len(almacenamiento.subidos) != 0 {
		t.Fatalf("creado=%v logo=%v subidos=%v", repo.creado, repo.logoCreado, almacenamiento.subidos)
	}
}

func TestCrearProyecto_PortadaInvalidaNoCreaElProyecto(t *testing.T) {
	casos := []struct {
		nombre  string
		archivo *ArchivoImagen
		err     error
	}{
		{"formato no permitido", imagen("tapa.gif", 1024), ErrProyectoLogoFormatoInvalido},
		{"sin extensión", imagen("tapa", 1024), ErrProyectoLogoFormatoInvalido},
		{"más de 5 MB", imagen("tapa.jpg", TamanoMaximoLogoProyecto+1), ErrProyectoLogoDemasiadoGrande},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			repo, almacenamiento, servicio := nuevoServicioPortada(nil)

			_, err := servicio.CrearProyecto(1, pedidoAltaProyecto(), caso.archivo)

			if !errors.Is(err, caso.err) {
				t.Fatalf("err = %v, se esperaba %v", err, caso.err)
			}
			if repo.crearLlamado || len(almacenamiento.subidos) != 0 {
				t.Fatal("con una portada inválida no se tiene que crear nada")
			}
		})
	}
}

func TestCrearProyecto_FallaLaSubidaNoDejaElProyectoCreado(t *testing.T) {
	repo, almacenamiento, servicio := nuevoServicioPortada(nil)
	almacenamiento.errSubir = errors.New("minio caído")

	_, err := servicio.CrearProyecto(1, pedidoAltaProyecto(), imagen("tapa.webp", 1024))

	if !errors.Is(err, ErrProyectoErrorAlmacenamiento) {
		t.Fatalf("err = %v", err)
	}
	if repo.creado {
		t.Fatal("si falla la subida el proyecto no tiene que quedar creado")
	}
}

func TestCrearProyecto_NombreVacioNoValidaNiSube(t *testing.T) {
	repo, _, servicio := nuevoServicioPortada(nil)
	pedido := pedidoAltaProyecto()
	pedido.Nombre = "   "

	_, err := servicio.CrearProyecto(1, pedido, imagen("tapa.png", 1024))

	if !errors.Is(err, ErrNombreProyectoObligatorio) || repo.crearLlamado {
		t.Fatalf("err = %v, crear llamado = %v", err, repo.crearLlamado)
	}
}

func TestEditarProyecto_PortadaNuevaReemplazaLaAnterior(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)

	respuesta, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("tapa.png", 1024), false)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.logoActualizado == nil || *repo.logoActualizado != anterior {
		t.Fatalf("logo guardado = %v", repo.logoActualizado)
	}
	// Mismo formato: la subida pisó el archivo, no hay que borrarlo.
	if len(almacenamiento.eliminados) != 0 {
		t.Fatalf("no se tenía que borrar nada: %v", almacenamiento.eliminados)
	}
	if respuesta.LogoUrl == nil || *respuesta.LogoUrl != "https://storage.test/"+anterior {
		t.Fatalf("logoUrl = %v", respuesta.LogoUrl)
	}
}

func TestEditarProyecto_CambioDeFormatoBorraElArchivoAnterior(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)

	if _, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("tapa.jpg", 1024), false); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.logoActualizado == nil || *repo.logoActualizado != "proyectos/9/logo.jpg" {
		t.Fatalf("logo guardado = %v", repo.logoActualizado)
	}
	if len(almacenamiento.eliminados) != 1 || almacenamiento.eliminados[0] != anterior {
		t.Fatalf("eliminados = %v", almacenamiento.eliminados)
	}
}

func TestEditarProyecto_QuitarPortadaLimpiaLaFilaYBorraElArchivo(t *testing.T) {
	anterior := "proyectos/9/logo.webp"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)

	respuesta, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, nil, true)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !repo.actualizado || repo.logoActualizado != nil {
		t.Fatalf("logo guardado = %v", repo.logoActualizado)
	}
	if len(almacenamiento.eliminados) != 1 || almacenamiento.eliminados[0] != anterior {
		t.Fatalf("eliminados = %v", almacenamiento.eliminados)
	}
	if respuesta.LogoUrl != nil {
		t.Fatalf("sin portada no tiene que haber logoUrl: %v", *respuesta.LogoUrl)
	}
}

func TestEditarProyecto_SinPedirQuitarConservaLaPortada(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)

	if _, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, nil, false); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.logoActualizado == nil || *repo.logoActualizado != anterior || len(almacenamiento.eliminados) != 0 {
		t.Fatalf("logo = %v, eliminados = %v", repo.logoActualizado, almacenamiento.eliminados)
	}
}

func TestEditarProyecto_PortadaNuevaTienePrioridadSobreQuitar(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, _, servicio := nuevoServicioPortada(&anterior)

	if _, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("tapa.webp", 1024), true); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.logoActualizado == nil || *repo.logoActualizado != "proyectos/9/logo.webp" {
		t.Fatalf("logo guardado = %v", repo.logoActualizado)
	}
}

func TestEditarProyecto_QuitarSinPortadaNoBorraNada(t *testing.T) {
	_, almacenamiento, servicio := nuevoServicioPortada(nil)

	if _, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, nil, true); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(almacenamiento.eliminados) != 0 {
		t.Fatalf("eliminados = %v", almacenamiento.eliminados)
	}
}

func TestEditarProyecto_PortadaInvalidaNoTocaNada(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)

	_, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("tapa.bmp", 1024), false)

	if !errors.Is(err, ErrProyectoLogoFormatoInvalido) {
		t.Fatalf("err = %v", err)
	}
	if repo.actualizado || len(almacenamiento.subidos) != 0 || len(almacenamiento.eliminados) != 0 {
		t.Fatal("una portada inválida no tiene que modificar nada")
	}
}

func TestEditarProyecto_FallaLaSubidaConservaLaPortada(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)
	almacenamiento.errSubir = errors.New("minio caído")

	_, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("tapa.jpg", 1024), false)

	if !errors.Is(err, ErrProyectoErrorAlmacenamiento) {
		t.Fatalf("err = %v", err)
	}
	if repo.actualizado || len(almacenamiento.eliminados) != 0 {
		t.Fatal("si falla la subida no se tiene que tocar la portada actual")
	}
}

func TestEditarProyecto_SoloElPropietarioCambiaLaPortada(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)
	repo.propietario = false

	_, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, nil, true)

	if !errors.Is(err, ErrProyectoSoloPropietario) {
		t.Fatalf("err = %v", err)
	}
	if repo.actualizado || len(almacenamiento.eliminados) != 0 {
		t.Fatal("un integrante que no es propietario no puede tocar la portada")
	}
}
