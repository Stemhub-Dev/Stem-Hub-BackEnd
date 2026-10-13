package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
)

const (
	permisoGestionarStems        = "GESTIONAR_STEMS"
	largoMaximoNombreStem        = 150
	largoMaximoNombreCategoria   = 100
	largoMaximoNombreArchivoStem = 255
)

var (
	ErrStemProyectoNoEncontrado = errors.New("el proyecto no existe")
	ErrStemVersionNoEncontrada  = errors.New("la versión no existe en la canción")
	ErrStemNoEncontrado         = errors.New("el stem no existe en la versión")
	ErrStemSinAcceso            = errors.New("el usuario no pertenece al proyecto")
	ErrStemSinPermiso           = errors.New("el usuario no puede gestionar stems en este proyecto")

	ErrStemNombreObligatorio = errors.New("el nombre del stem es obligatorio")
	ErrStemNombreLargo       = errors.New("el nombre del stem es demasiado largo")
	ErrStemNombreDuplicado   = errors.New("ya existe un stem con ese nombre en la versión")

	ErrStemNombreArchivoObligatorio = errors.New("el nombre del archivo del stem es obligatorio")
	ErrStemNombreArchivoLargo       = errors.New("el nombre del archivo del stem es demasiado largo")
	ErrStemNombreArchivoInvalido    = errors.New("el nombre del archivo del stem no puede contener barras")

	ErrStemCategoriaObligatoria   = errors.New("la categoría del stem es obligatoria")
	ErrStemCategoriaNoEncontrada  = errors.New("la categoría no existe")
	ErrStemCategoriaDuplicada     = errors.New("ya existe una categoría con ese nombre")
	ErrStemCategoriaNombreLargo   = errors.New("el nombre de la categoría es demasiado largo")
	ErrStemArchivoObligatorio     = errors.New("el archivo del stem es obligatorio")
	ErrStemArchivoVacio           = errors.New("el archivo del stem está vacío")
	ErrStemFormatoInvalido        = errors.New("el formato del stem no está soportado")
	ErrStemArchivoDemasiadoGrande = errors.New("el stem supera el tamaño máximo permitido")
	ErrStemPerfilRequerido        = errors.New("el usuario no posee un perfil de integrante")
)

// CrearStemRequest: la categoría es una existente (CodCategoriaStem) o una
// nueva del proyecto (NuevaCategoria), nunca ambas.
type CrearStemRequest struct {
	Nombre           string
	CodCategoriaStem *int64
	NuevaCategoria   *string
}

// EditarStemRequest: el nombre es obligatorio; NombreArchivo y la categoría
// son opcionales (nil = se conservan). NombreArchivo va sin extensión: la
// pone el service según el formato real del archivo. La categoría sigue las
// reglas del alta. La canción y la versión del stem no se pueden cambiar.
type EditarStemRequest struct {
	Nombre           string
	NombreArchivo    *string
	CodCategoriaStem *int64
	NuevaCategoria   *string
}

type StemService interface {
	ListarCategorias(
		codigoUsuario int64,
		codigoProyecto int64,
	) ([]dto.CategoriaStemResponse, error)

	Listar(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
	) ([]dto.StemListadoResponse, error)

	Crear(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		request CrearStemRequest,
		archivo ArchivoAudio,
	) (*dto.StemListadoResponse, error)

	// archivo nil = se conserva el actual.
	Editar(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		codStem int64,
		request EditarStemRequest,
		archivo *ArchivoAudio,
	) (*dto.StemListadoResponse, error)

	Eliminar(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		codStem int64,
	) error

	ObtenerURLAudio(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
		codStem int64,
	) (*dto.AudioStemResponse, error)

	ListarGlobal(
		codigoUsuario int64,
		filtro dto.ListarStemsGlobalFiltro,
	) (*dto.StemsGlobalesPaginadosResponse, error)
}

type stemService struct {
	stemRepository       repository.StemRepository
	cancionRepository    repository.CancionRepository
	proyectoRepository   repository.ProyectoRepository
	integranteRepository repository.IntegranteRepository
	audioStorage         storage.AudioStorage
}

func NewStemService(
	stemRepository repository.StemRepository,
	cancionRepository repository.CancionRepository,
	proyectoRepository repository.ProyectoRepository,
	integranteRepository repository.IntegranteRepository,
	audioStorage storage.AudioStorage,
) StemService {

	return &stemService{
		stemRepository:       stemRepository,
		cancionRepository:    cancionRepository,
		proyectoRepository:   proyectoRepository,
		integranteRepository: integranteRepository,
		audioStorage:         audioStorage,
	}
}

func stemAResponse(stem model.Stem) dto.StemListadoResponse {
	return dto.StemListadoResponse{
		CodStem:          stem.CodStem,
		Nombre:           stem.NombreStem,
		CodCategoriaStem: stem.CodCategoriaStem,
		NombreCategoria:  stem.NombreCategoriaStem,
		GeneradoConIA:    stem.GeneradoConIA,
		NombreArchivo:    stem.NombreArchivoStem,
		FormatoArchivo:   stem.FormatoArchivoStem,
	}
}

// La key lleva un sufijo aleatorio y no el codstem: el archivo se sube
// antes de insertar la fila (la fila exige la key), y al reemplazarlo el
// nuevo convive un momento con el viejo hasta que se borra.
func claveObjetoStem(codigoProyecto, codigoCancion, codigoVersion int64, formato string) (string, error) {

	sufijo := make([]byte, 8)

	if _, err := rand.Read(sufijo); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"proyectos/%d/canciones/%d/versiones/%d/stems/%s.%s",
		codigoProyecto,
		codigoCancion,
		codigoVersion,
		hex.EncodeToString(sufijo),
		formato,
	), nil
}

// validarAcceso comprueba que la versión exista dentro de la canción y el
// proyecto, y que el usuario pertenezca al proyecto. Con permiso != "" exige
// además ese permiso en su rol del proyecto.
func (s *stemService) validarAcceso(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	permiso string,
) error {

	_, err := validarAccesoVersionStem(
		s.proyectoRepository,
		s.cancionRepository,
		s.integranteRepository,
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
		permiso,
	)

	return err
}

func (s *stemService) validarAccesoProyecto(
	codigoUsuario int64,
	codigoProyecto int64,
	permiso string,
) error {

	_, err := validarAccesoProyectoStem(
		s.proyectoRepository,
		s.integranteRepository,
		codigoUsuario,
		codigoProyecto,
		permiso,
	)

	return err
}

// validarAccesoVersionStem es validarAcceso como función del paquete, para
// reusarla desde la separación de stems; devuelve el integrante del usuario.
func validarAccesoVersionStem(
	proyectoRepository repository.ProyectoRepository,
	cancionRepository repository.CancionRepository,
	integranteRepository repository.IntegranteRepository,
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	permiso string,
) (*model.Integrante, error) {

	integrante, err := validarAccesoProyectoStem(
		proyectoRepository,
		integranteRepository,
		codigoUsuario,
		codigoProyecto,
		permiso,
	)

	if err != nil {
		return nil, err
	}

	existeCancion, err :=
		cancionRepository.ExisteCancionActivaEnProyecto(
			codigoProyecto,
			codigoCancion,
		)

	if err != nil {
		return nil, err
	}

	if !existeCancion {
		return nil, ErrStemVersionNoEncontrada
	}

	existeVersion, err :=
		cancionRepository.ExisteVersionActivaEnCancion(
			codigoCancion,
			codigoVersion,
		)

	if err != nil {
		return nil, err
	}

	if !existeVersion {
		return nil, ErrStemVersionNoEncontrada
	}

	return integrante, nil
}

func validarAccesoProyectoStem(
	proyectoRepository repository.ProyectoRepository,
	integranteRepository repository.IntegranteRepository,
	codigoUsuario int64,
	codigoProyecto int64,
	permiso string,
) (*model.Integrante, error) {

	existeProyecto, err :=
		proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrStemProyectoNoEncontrado
	}

	integrante, err :=
		integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	esIntegrante, err :=
		proyectoRepository.EsIntegranteActivo(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !esIntegrante {
		return nil, ErrStemSinAcceso
	}

	if permiso == "" {
		return integrante, nil
	}

	puede, err :=
		proyectoRepository.PuedeRealizarEnProyecto(
			integrante.CodIntegrante,
			codigoProyecto,
			permiso,
		)

	if err != nil {
		return nil, err
	}

	if !puede {
		return nil, ErrStemSinPermiso
	}

	return integrante, nil
}

func validarNombreStem(nombre string) (string, error) {

	nombre = strings.TrimSpace(nombre)

	if nombre == "" {
		return "", ErrStemNombreObligatorio
	}

	if utf8.RuneCountInString(nombre) > largoMaximoNombreStem {
		return "", ErrStemNombreLargo
	}

	return nombre, nil
}

// validarNuevaCategoria recorta el nombre de la categoría a crear; nil si
// se eligió una existente.
func validarNuevaCategoria(nuevaCategoria *string) (*string, error) {

	if nuevaCategoria == nil {
		return nil, nil
	}

	categoria := strings.TrimSpace(*nuevaCategoria)

	if categoria == "" {
		return nil, ErrStemCategoriaObligatoria
	}

	if utf8.RuneCountInString(categoria) > largoMaximoNombreCategoria {
		return nil, ErrStemCategoriaNombreLargo
	}

	return &categoria, nil
}

// resolverCategoria devuelve el código de la categoría existente elegida,
// o 0 si se va a crear una nueva (que no puede repetir el nombre de otra).
func (s *stemService) resolverCategoria(
	codigoProyecto int64,
	codCategoriaStem *int64,
	nuevaCategoria *string,
) (int64, error) {

	if nuevaCategoria != nil {
		existe, err := s.stemRepository.ExisteCategoriaConNombre(codigoProyecto, *nuevaCategoria)

		if err != nil {
			return 0, err
		}

		if existe {
			return 0, ErrStemCategoriaDuplicada
		}

		return 0, nil
	}

	categoria, err := s.stemRepository.BuscarCategoria(codigoProyecto, *codCategoriaStem)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrStemCategoriaNoEncontrada
	}

	if err != nil {
		return 0, err
	}

	return categoria.CodCategoriaStem, nil
}

// nombreArchivoStem arma el nombre de archivo que se muestra del stem: el
// que eligió el usuario más la extensión del formato real. Si el usuario ya
// la escribió, no se repite. Es solo el nombre visible: la key en MinIO no
// cambia.
func nombreArchivoStem(nombre string, formato string) (string, error) {

	nombre = strings.TrimSpace(nombre)
	extension := "." + formato

	if len(nombre) >= len(extension) &&
		strings.EqualFold(nombre[len(nombre)-len(extension):], extension) {
		nombre = strings.TrimSpace(nombre[:len(nombre)-len(extension)])
	}

	if nombre == "" {
		return "", ErrStemNombreArchivoObligatorio
	}

	if strings.ContainsAny(nombre, `/\`) {
		return "", ErrStemNombreArchivoInvalido
	}

	nombre += extension

	if utf8.RuneCountInString(nombre) > largoMaximoNombreArchivoStem {
		return "", ErrStemNombreArchivoLargo
	}

	return nombre, nil
}

// validarArchivoStem aplica las mismas reglas de formato que la pista de
// una canción, con su propio límite de tamaño.
func validarArchivoStem(archivo ArchivoAudio) (string, error) {

	if archivo.Contenido == nil {
		return "", ErrStemArchivoObligatorio
	}

	if archivo.Tamano == 0 {
		return "", ErrStemArchivoVacio
	}

	if archivo.Tamano > TamanoMaximoArchivoStem {
		return "", ErrStemArchivoDemasiadoGrande
	}

	formato, err := formatoDesdeNombreArchivo(archivo.NombreOriginal)

	if err != nil {
		return "", ErrStemFormatoInvalido
	}

	return formato, nil
}

func (s *stemService) subirArchivo(
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	archivo ArchivoAudio,
	formato string,
) (*repository.ArchivoStemGuardado, error) {

	objectKey, err := claveObjetoStem(codigoProyecto, codigoCancion, codigoVersion, formato)

	if err != nil {
		return nil, err
	}

	if err := s.audioStorage.Subir(
		context.Background(),
		objectKey,
		archivo.Contenido,
		archivo.Tamano,
		formatosAudioPermitidos[formato],
	); err != nil {
		return nil, ErrCancionErrorAlmacenamiento
	}

	return &repository.ArchivoStemGuardado{
		URL:            objectKey,
		Formato:        formato,
		NombreOriginal: archivo.NombreOriginal,
	}, nil
}

// Best-effort: si falla queda un objeto huérfano en MinIO, pero la
// operación del usuario ya se completó y no tiene sentido revertirla.
func (s *stemService) eliminarArchivo(objectKey string) {
	if err := s.audioStorage.Eliminar(context.Background(), objectKey); err != nil {
		log.Println("No se pudo eliminar el archivo del stem en MinIO:", objectKey, err)
	}
}

func (s *stemService) ListarCategorias(
	codigoUsuario int64,
	codigoProyecto int64,
) ([]dto.CategoriaStemResponse, error) {

	if err := s.validarAccesoProyecto(codigoUsuario, codigoProyecto, ""); err != nil {
		return nil, err
	}

	categorias, err := s.stemRepository.ListarCategorias(codigoProyecto)

	if err != nil {
		return nil, err
	}

	respuesta := make([]dto.CategoriaStemResponse, 0, len(categorias))

	for _, categoria := range categorias {
		respuesta = append(respuesta, dto.CategoriaStemResponse{
			CodCategoriaStem: categoria.CodCategoriaStem,
			Nombre:           categoria.NombreCategoriaStem,
			PorDefecto:       categoria.CodigoProyecto == nil,
		})
	}

	return respuesta, nil
}

func (s *stemService) Listar(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
) ([]dto.StemListadoResponse, error) {

	if err := s.validarAcceso(codigoUsuario, codigoProyecto, codigoCancion, codigoVersion, ""); err != nil {
		return nil, err
	}

	stems, err := s.stemRepository.ListarPorVersion(codigoVersion)

	if err != nil {
		return nil, err
	}

	respuesta := make([]dto.StemListadoResponse, 0, len(stems))

	for _, stem := range stems {
		respuesta = append(respuesta, stemAResponse(stem))
	}

	return respuesta, nil
}

func (s *stemService) Crear(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	request CrearStemRequest,
	archivo ArchivoAudio,
) (*dto.StemListadoResponse, error) {

	// Validaciones de datos antes que las de acceso, igual que en canciones:
	// no hace falta ir a la base para rechazar un formulario incompleto.
	nombre, err := validarNombreStem(request.Nombre)

	if err != nil {
		return nil, err
	}

	nuevaCategoria, err := validarNuevaCategoria(request.NuevaCategoria)

	if err != nil {
		return nil, err
	}

	if nuevaCategoria == nil && request.CodCategoriaStem == nil {
		return nil, ErrStemCategoriaObligatoria
	}

	formato, err := validarArchivoStem(archivo)

	if err != nil {
		return nil, err
	}

	if err := s.validarAcceso(
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
		permisoGestionarStems,
	); err != nil {
		return nil, err
	}

	codCategoriaStem, err := s.resolverCategoria(codigoProyecto, request.CodCategoriaStem, nuevaCategoria)

	if err != nil {
		return nil, err
	}

	existeNombre, err := s.stemRepository.ExisteNombreEnVersion(codigoVersion, nombre, 0)

	if err != nil {
		return nil, err
	}

	if existeNombre {
		return nil, ErrStemNombreDuplicado
	}

	guardado, err := s.subirArchivo(codigoProyecto, codigoCancion, codigoVersion, archivo, formato)

	if err != nil {
		return nil, err
	}

	codStem, err := s.stemRepository.Crear(
		codigoProyecto,
		codigoVersion,
		nombre,
		codCategoriaStem,
		nuevaCategoria,
		*guardado,
	)

	if err != nil {
		s.eliminarArchivo(guardado.URL)
		return nil, err
	}

	stem, err := s.stemRepository.BuscarPorCodigo(codigoVersion, codStem)

	if err != nil {
		return nil, err
	}

	respuesta := stemAResponse(*stem)

	return &respuesta, nil
}

func (s *stemService) buscarStem(codigoVersion, codStem int64) (*model.Stem, error) {

	stem, err := s.stemRepository.BuscarPorCodigo(codigoVersion, codStem)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrStemNoEncontrado
	}

	return stem, err
}

func (s *stemService) Editar(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	codStem int64,
	request EditarStemRequest,
	archivo *ArchivoAudio,
) (*dto.StemListadoResponse, error) {

	nombre, err := validarNombreStem(request.Nombre)

	if err != nil {
		return nil, err
	}

	nuevaCategoria, err := validarNuevaCategoria(request.NuevaCategoria)

	if err != nil {
		return nil, err
	}

	var formato string

	if archivo != nil {
		if formato, err = validarArchivoStem(*archivo); err != nil {
			return nil, err
		}
	}

	if err := s.validarAcceso(
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
		permisoGestionarStems,
	); err != nil {
		return nil, err
	}

	actual, err := s.buscarStem(codigoVersion, codStem)

	if err != nil {
		return nil, err
	}

	existeNombre, err := s.stemRepository.ExisteNombreEnVersion(codigoVersion, nombre, codStem)

	if err != nil {
		return nil, err
	}

	if existeNombre {
		return nil, ErrStemNombreDuplicado
	}

	// Elegir la categoría que ya tiene no la vuelve a validar: puede ser una
	// que se dio de baja después de cargar el stem.
	codCategoriaStem := actual.CodCategoriaStem
	cambiaCategoria := nuevaCategoria != nil ||
		(request.CodCategoriaStem != nil && *request.CodCategoriaStem != actual.CodCategoriaStem)

	if cambiaCategoria {
		if codCategoriaStem, err = s.resolverCategoria(codigoProyecto, request.CodCategoriaStem, nuevaCategoria); err != nil {
			return nil, err
		}
	}

	// Sin nombre de archivo explícito: el del archivo nuevo si se reemplaza,
	// si no el actual.
	nombreArchivo := actual.NombreArchivoStem

	if archivo != nil {
		nombreArchivo = archivo.NombreOriginal
	} else {
		formato = actual.FormatoArchivoStem
	}

	if request.NombreArchivo != nil {
		if nombreArchivo, err = nombreArchivoStem(*request.NombreArchivo, formato); err != nil {
			return nil, err
		}
	}

	var guardado *repository.ArchivoStemGuardado

	if archivo != nil {
		if guardado, err = s.subirArchivo(codigoProyecto, codigoCancion, codigoVersion, *archivo, formato); err != nil {
			return nil, err
		}
	}

	if err := s.stemRepository.Actualizar(codigoProyecto, codStem, repository.CambiosStem{
		Nombre:           nombre,
		NombreArchivo:    nombreArchivo,
		CodCategoriaStem: codCategoriaStem,
		NuevaCategoria:   nuevaCategoria,
		Archivo:          guardado,
	}); err != nil {
		if guardado != nil {
			s.eliminarArchivo(guardado.URL)
		}
		return nil, err
	}

	// Borrado real también al reemplazar: el archivo anterior no se conserva.
	if guardado != nil {
		s.eliminarArchivo(actual.URLArchivoStem)
	}

	stem, err := s.stemRepository.BuscarPorCodigo(codigoVersion, codStem)

	if err != nil {
		return nil, err
	}

	respuesta := stemAResponse(*stem)

	return &respuesta, nil
}

func (s *stemService) Eliminar(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	codStem int64,
) error {

	if err := s.validarAcceso(
		codigoUsuario,
		codigoProyecto,
		codigoCancion,
		codigoVersion,
		permisoGestionarStems,
	); err != nil {
		return err
	}

	stem, err := s.buscarStem(codigoVersion, codStem)

	if err != nil {
		return err
	}

	if err := s.stemRepository.Eliminar(codStem); err != nil {
		return err
	}

	s.eliminarArchivo(stem.URLArchivoStem)

	return nil
}

func (s *stemService) ObtenerURLAudio(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
	codStem int64,
) (*dto.AudioStemResponse, error) {

	if err := s.validarAcceso(codigoUsuario, codigoProyecto, codigoCancion, codigoVersion, ""); err != nil {
		return nil, err
	}

	stem, err := s.buscarStem(codigoVersion, codStem)

	if err != nil {
		return nil, err
	}

	url, err := s.audioStorage.ObtenerURLDescarga(
		context.Background(),
		stem.URLArchivoStem,
		VigenciaURLDescargaAudio,
	)

	if err != nil {
		return nil, err
	}

	return &dto.AudioStemResponse{
		CodStem:          codStem,
		URL:              url,
		ExpiraEnSegundos: int(VigenciaURLDescargaAudio.Seconds()),
	}, nil
}

func (s *stemService) ListarGlobal(
	codigoUsuario int64,
	filtro dto.ListarStemsGlobalFiltro,
) (*dto.StemsGlobalesPaginadosResponse, error) {

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrStemPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	if integrante.FechaHoraBajaIntegrante != nil {
		return nil,
			ErrStemPerfilRequerido
	}

	filtro.Busqueda =
		strings.TrimSpace(
			filtro.Busqueda,
		)

	total, err :=
		s.stemRepository.
			ContarGlobalPorProductor(
				integrante.CodIntegrante,
				filtro,
			)

	if err != nil {
		return nil, err
	}

	totalPaginas :=
		(total +
			filtro.TamanoPagina -
			1) /
			filtro.TamanoPagina

	respuesta :=
		&dto.StemsGlobalesPaginadosResponse{
			Data: make(
				[]dto.StemGlobalListadoResponse,
				0,
			),

			TotalItems: total,

			TotalPages: totalPaginas,

			CurrentPage: filtro.Pagina,
		}

	// Si no existen stems o la página solicitada
	// supera la última página, respondemos 200 con data vacía.
	if total == 0 ||
		filtro.Pagina > totalPaginas {

		return respuesta, nil
	}

	desplazamiento :=
		(filtro.Pagina - 1) *
			filtro.TamanoPagina

	stems, err :=
		s.stemRepository.
			ListarGlobalPorProductor(
				integrante.CodIntegrante,
				filtro,
				desplazamiento,
			)

	if err != nil {
		return nil, err
	}

	respuesta.Data =
		append(
			respuesta.Data,
			stems...,
		)

	return respuesta, nil
}
