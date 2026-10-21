package service

import (
	"errors"
	"fmt"
	"regexp"
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

	// Fallo después de subir el logo (UPDATE o COMMIT).
	errTrasSubir error

	// Logo que otra edición guardó entre la lectura del service y la
	// transacción de Actualizar.
	logoCambiadoPorOtro *string

	crearLlamado     bool
	creado           bool
	logoCreado       *string
	actualizado      bool
	cambioLogo       bool
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

	if r.errTrasSubir != nil {
		return 0, r.errTrasSubir
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
	cambiarLogo bool,
	logoObjectKey *string,
	_ int64,
	_ int64,
	_ []int64,
) (*string, *string, error) {
	if r.errTrasSubir != nil {
		return nil, nil, r.errTrasSubir
	}

	r.actualizado = true
	r.cambioLogo = cambiarLogo

	if r.logoCambiadoPorOtro != nil {
		r.logoActual = r.logoCambiadoPorOtro
	}

	if !cambiarLogo {
		return r.logoActual, nil, nil
	}

	anterior := r.logoActual
	r.logoActualizado = logoObjectKey
	r.logoActual = logoObjectKey

	return logoObjectKey, anterior, nil
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

// esClaveLogo verifica proyectos/{id}/logo-{uuid}.{formato}.
func esClaveLogo(t *testing.T, clave *string, formato string) {
	t.Helper()

	patron := regexp.MustCompile(`^proyectos/9/logo-[0-9a-f-]{36}\.` + formato + `$`)
	if clave == nil || !patron.MatchString(*clave) {
		t.Fatalf("clave del logo = %v, se esperaba proyectos/9/logo-<uuid>.%s", clave, formato)
	}
}

func TestCrearProyecto_ConPortadaLaSubeConLaClaveDelProyecto(t *testing.T) {
	repo, almacenamiento, servicio := nuevoServicioPortada(nil)

	_, err := servicio.CrearProyecto(1, pedidoAltaProyecto(), imagen("tapa.PNG", 1024))

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	esClaveLogo(t, repo.logoCreado, "png")
	if len(almacenamiento.subidos) != 1 || almacenamiento.subidos[0] != *repo.logoCreado {
		t.Fatalf("subidos = %v, guardado = %v", almacenamiento.subidos, *repo.logoCreado)
	}
}

func TestCrearProyecto_CadaSubidaUsaUnaClaveNueva(t *testing.T) {
	primero, _, servicio := nuevoServicioPortada(nil)
	if _, err := servicio.CrearProyecto(1, pedidoAltaProyecto(), imagen("tapa.png", 1024)); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	segundo, _, servicio := nuevoServicioPortada(nil)
	if _, err := servicio.CrearProyecto(1, pedidoAltaProyecto(), imagen("tapa.png", 1024)); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if *primero.logoCreado == *segundo.logoCreado {
		t.Fatalf("dos subidas con la misma clave: %s", *primero.logoCreado)
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

func TestCrearProyecto_SiFallaLaBaseDespuesDeSubirBorraElLogo(t *testing.T) {
	repo, almacenamiento, servicio := nuevoServicioPortada(nil)
	errBase := errors.New("fallo el UPDATE del logo")
	repo.errTrasSubir = errBase

	_, err := servicio.CrearProyecto(1, pedidoAltaProyecto(), imagen("tapa.png", 1024))

	if !errors.Is(err, errBase) {
		t.Fatalf("err = %v", err)
	}
	if len(almacenamiento.eliminados) != 1 || almacenamiento.eliminados[0] != *repo.logoCreado {
		t.Fatalf("eliminados = %v, subido = %v", almacenamiento.eliminados, *repo.logoCreado)
	}
}

func TestCrearProyecto_SiElCommitEsInciertoNoBorraElLogo(t *testing.T) {
	repo, almacenamiento, servicio := nuevoServicioPortada(nil)
	repo.errTrasSubir = fmt.Errorf("%w: conexión cortada", repository.ErrConfirmacionIncierta)

	if _, err := servicio.CrearProyecto(1, pedidoAltaProyecto(), imagen("tapa.png", 1024)); err == nil {
		t.Fatal("se esperaba error")
	}
	if len(almacenamiento.eliminados) != 0 {
		t.Fatalf("con un commit incierto el logo puede estar en uso: %v", almacenamiento.eliminados)
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

func TestEditarProyecto_PortadaNuevaReemplazaYBorraLaAnterior(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)

	respuesta, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("tapa.png", 1024), false)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !repo.cambioLogo {
		t.Fatal("se tenía que pedir el cambio de logo")
	}
	esClaveLogo(t, repo.logoActualizado, "png")
	if len(almacenamiento.eliminados) != 1 || almacenamiento.eliminados[0] != anterior {
		t.Fatalf("eliminados = %v", almacenamiento.eliminados)
	}
	if respuesta.LogoUrl == nil || *respuesta.LogoUrl != "https://storage.test/"+*repo.logoActualizado {
		t.Fatalf("logoUrl = %v", respuesta.LogoUrl)
	}
}

func TestEditarProyecto_QuitarPortadaLimpiaLaFilaYBorraElArchivo(t *testing.T) {
	anterior := "proyectos/9/logo.webp"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)

	respuesta, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, nil, true)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !repo.cambioLogo || repo.logoActualizado != nil {
		t.Fatalf("cambio = %v, logo guardado = %v", repo.cambioLogo, repo.logoActualizado)
	}
	if len(almacenamiento.eliminados) != 1 || almacenamiento.eliminados[0] != anterior {
		t.Fatalf("eliminados = %v", almacenamiento.eliminados)
	}
	if respuesta.LogoUrl != nil {
		t.Fatalf("sin portada no tiene que haber logoUrl: %v", *respuesta.LogoUrl)
	}
}

func TestEditarProyecto_SinTocarLaPortadaNoLaReescribe(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)

	respuesta, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, nil, false)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	// Reescribir el logo leído antes podría pisar el de una edición
	// simultánea: sin cambio, el repository conserva el de la fila.
	if !repo.actualizado || repo.cambioLogo || len(almacenamiento.eliminados) != 0 {
		t.Fatalf("cambio = %v, eliminados = %v", repo.cambioLogo, almacenamiento.eliminados)
	}
	if respuesta.LogoUrl == nil || *respuesta.LogoUrl != "https://storage.test/"+anterior {
		t.Fatalf("logoUrl = %v", respuesta.LogoUrl)
	}
}

func TestEditarProyecto_PortadaNuevaTienePrioridadSobreQuitar(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, _, servicio := nuevoServicioPortada(&anterior)

	if _, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("tapa.webp", 1024), true); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	esClaveLogo(t, repo.logoActualizado, "webp")
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

// Dos ediciones seguidas: cada una borra solo el archivo que reemplazó, que
// es el que le devuelve el repository, no el que leyó al empezar.
func TestEditarProyecto_CadaEdicionBorraSoloLoQueReemplazo(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)

	if _, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("a.png", 1024), false); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	primera := *repo.logoActualizado

	if _, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("b.png", 1024), false); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if len(almacenamiento.eliminados) != 2 ||
		almacenamiento.eliminados[0] != anterior ||
		almacenamiento.eliminados[1] != primera {
		t.Fatalf("eliminados = %v", almacenamiento.eliminados)
	}
	if *repo.logoActualizado == primera {
		t.Fatal("la segunda edición tenía que usar una clave nueva")
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

func TestEditarProyecto_SiFallaLaBaseBorraElLogoNuevoYConservaElAnterior(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)
	repo.errTrasSubir = errors.New("fallo el UPDATE")

	if _, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("tapa.jpg", 1024), false); err == nil {
		t.Fatal("se esperaba error")
	}

	if len(almacenamiento.subidos) != 1 ||
		len(almacenamiento.eliminados) != 1 ||
		almacenamiento.eliminados[0] != almacenamiento.subidos[0] {
		t.Fatalf("subidos = %v, eliminados = %v", almacenamiento.subidos, almacenamiento.eliminados)
	}
}

func TestEditarProyecto_SiElCommitEsInciertoNoBorraNada(t *testing.T) {
	anterior := "proyectos/9/logo.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&anterior)
	repo.errTrasSubir = fmt.Errorf("%w: conexión cortada", repository.ErrConfirmacionIncierta)

	if _, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, imagen("tapa.jpg", 1024), false); err == nil {
		t.Fatal("se esperaba error")
	}
	if len(almacenamiento.eliminados) != 0 {
		t.Fatalf("con un commit incierto no se sabe qué logo quedó: %v", almacenamiento.eliminados)
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

// Una edición sin cambio de portada leyó la portada A, pero antes de que
// guarde otra edición la reemplazó por B (y borró A): la respuesta tiene que
// firmar B, que es la que quedó en la fila.
func TestEditarProyecto_SinTocarLaPortadaRespondeLaVigente(t *testing.T) {
	leida := "proyectos/9/logo-a.png"
	vigente := "proyectos/9/logo-b.png"
	repo, almacenamiento, servicio := nuevoServicioPortada(&leida)
	repo.logoCambiadoPorOtro = &vigente

	respuesta, err := servicio.EditarProyecto(1, 9, dto.EditarProyectoRequest{}, nil, false)

	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if respuesta.LogoUrl == nil || *respuesta.LogoUrl != "https://storage.test/"+vigente {
		t.Fatalf("logoUrl = %v, se esperaba la portada vigente", respuesta.LogoUrl)
	}
	if len(almacenamiento.eliminados) != 0 {
		t.Fatalf("sin cambio de portada no se borra nada: %v", almacenamiento.eliminados)
	}
}
