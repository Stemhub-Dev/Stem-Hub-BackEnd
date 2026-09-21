package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/dto"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
)

const TamanoMaximoArchivoAudio int64 = 100 * 1024 * 1024 // 100 MB

var formatosAudioPermitidos = map[string]string{
	"mp3":  "audio/mpeg",
	"wav":  "audio/wav",
	"flac": "audio/flac",
}

var (
	ErrCancionNombreObligatorio = errors.New(
		"el nombre de la canción es obligatorio",
	)

	ErrCancionPistaObligatoria = errors.New(
		"la pista inicial es obligatoria",
	)

	ErrCancionProyectoNoEncontrado = errors.New(
		"el proyecto no existe",
	)

	ErrCancionSinPermiso = errors.New(
		"el usuario no puede crear canciones en este proyecto",
	)

	ErrCancionNombreDuplicado = errors.New(
		"ya existe una canción con ese nombre en el proyecto",
	)

	ErrCancionPerfilRequerido = errors.New(
		"el usuario no posee perfil",
	)

	ErrCancionFormatoInvalido = errors.New(
		"el formato del archivo de audio no está soportado",
	)

	ErrCancionArchivoDemasiadoGrande = errors.New(
		"el archivo de audio supera el tamaño máximo permitido",
	)

	ErrCancionErrorAlmacenamiento = errors.New(
		"error al almacenar el archivo de audio",
	)

	ErrVersionPistaObligatoria = errors.New(
		"la pista de la nueva versión es obligatoria",
	)

	ErrVersionCancionNoEncontrada = errors.New(
		"la canción no existe en el proyecto",
	)

	ErrVersionSinPermiso = errors.New(
		"el usuario no puede crear versiones en este proyecto",
	)

	ErrCancionSinAccesoProyecto = errors.New(
		"el usuario no pertenece al proyecto",
	)

	ErrVersionSinArchivo = errors.New(
		"la versión no tiene un archivo de audio cargado",
	)

	ErrStemNombreObligatorio = errors.New(
		"cada stem debe tener un nombre",
	)

	ErrStemFormatoInvalido = errors.New(
		"el formato de un stem no está soportado",
	)

	ErrStemArchivoDemasiadoGrande = errors.New(
		"un stem supera el tamaño máximo permitido",
	)

	ErrCancionNoEncontrada = errors.New(
		"la canción no existe en el proyecto",
	)
)

// VigenciaURLDescargaAudio es el tiempo de validez de la URL presignada
// devuelta para reproducir/descargar el audio de una versión.
const VigenciaURLDescargaAudio = 15 * time.Minute

// ArchivoAudio representa el archivo de audio recibido por el handler,
// desacoplado de multipart.FileHeader para no filtrar detalles de Gin al
// service.
type ArchivoAudio struct {
	Contenido      io.Reader
	NombreOriginal string
	Tamano         int64
}

// ArchivoStem representa un stem opcional recibido junto con una nueva
// versión: el nombre lo elige libremente quien sube el archivo (ej.
// "Batería", "Voz principal"), sin catálogo fijo.
type ArchivoStem struct {
	Nombre         string
	Contenido      io.Reader
	NombreOriginal string
	Tamano         int64
}

func formatoDesdeNombreArchivo(nombreArchivo string) (string, error) {
	partes := strings.Split(nombreArchivo, ".")

	if len(partes) < 2 {
		return "", ErrCancionFormatoInvalido
	}

	extension := strings.ToLower(partes[len(partes)-1])

	if _, ok := formatosAudioPermitidos[extension]; !ok {
		return "", ErrCancionFormatoInvalido
	}

	return extension, nil
}

type CancionService interface {
	Crear(
		codigoUsuario int64,
		codigoProyecto int64,
		nombre string,
		archivo ArchivoAudio,
	) (*dto.CrearCancionResponse, error)

	CrearVersion(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		archivo ArchivoAudio,
		notas *string,
		stems []ArchivoStem,
	) (*dto.CrearVersionCancionResponse, error)

	ListarPorProyecto(
		codigoUsuario int64,
		codigoProyecto int64,
	) ([]dto.CancionListadoResponse, error)

	ListarVersiones(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
	) ([]dto.VersionCancionListadoResponse, error)

	ObtenerURLDescargaVersion(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		codigoVersion int64,
	) (*dto.AudioVersionResponse, error)

	ListarMisCanciones(
		codigoUsuario int64,
	) ([]dto.MiCancionListadoResponse, error)

	Editar(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
		request dto.EditarCancionRequest,
	) (*dto.EditarCancionResponse, error)

	DarDeBaja(
		codigoUsuario int64,
		codigoProyecto int64,
		codigoCancion int64,
	) error
}

type cancionService struct {
	cancionRepository    repository.CancionRepository
	proyectoRepository   repository.ProyectoRepository
	integranteRepository repository.IntegranteRepository
	audioStorage         storage.AudioStorage
}

func NewCancionService(
	cancionRepository repository.CancionRepository,
	proyectoRepository repository.ProyectoRepository,
	integranteRepository repository.IntegranteRepository,
	audioStorage storage.AudioStorage,
) CancionService {

	return &cancionService{
		cancionRepository:    cancionRepository,
		proyectoRepository:   proyectoRepository,
		integranteRepository: integranteRepository,
		audioStorage:         audioStorage,
	}
}

func claveObjetoAudio(codigoProyecto, codigoCancion int64, numeroVersion int, formato string) string {
	return fmt.Sprintf(
		"proyectos/%d/canciones/%d/v%d.%s",
		codigoProyecto,
		codigoCancion,
		numeroVersion,
		formato,
	)
}

func claveObjetoStem(codigoProyecto, codigoCancion int64, numeroVersion int, indice int, formato string) string {
	return fmt.Sprintf(
		"proyectos/%d/canciones/%d/v%d/stems/%d.%s",
		codigoProyecto,
		codigoCancion,
		numeroVersion,
		indice,
		formato,
	)
}

func (s *cancionService) Crear(
	codigoUsuario int64,
	codigoProyecto int64,
	nombre string,
	archivo ArchivoAudio,
) (*dto.CrearCancionResponse, error) {

	nombre = strings.TrimSpace(nombre)

	if nombre == "" {
		return nil, ErrCancionNombreObligatorio
	}

	if archivo.Contenido == nil {
		return nil, ErrCancionPistaObligatoria
	}

	if archivo.Tamano > TamanoMaximoArchivoAudio {
		return nil, ErrCancionArchivoDemasiadoGrande
	}

	formato, err := formatoDesdeNombreArchivo(archivo.NombreOriginal)

	if err != nil {
		return nil, err
	}

	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrCancionProyectoNoEncontrado
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	puedeGestionar, err :=
		s.proyectoRepository.PuedeGestionarCanciones(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !puedeGestionar {
		return nil, ErrCancionSinPermiso
	}

	existeNombre, err :=
		s.cancionRepository.ExisteNombreEnProyecto(
			codigoProyecto,
			nombre,
		)

	if err != nil {
		return nil, err
	}

	if existeNombre {
		return nil, ErrCancionNombreDuplicado
	}

	tx, codigoCancion, err :=
		s.cancionRepository.IniciarCreacionCancion(
			codigoProyecto,
			nombre,
		)

	if err != nil {
		return nil, err
	}

	objectKey := claveObjetoAudio(codigoProyecto, codigoCancion, 1, formato)

	if err := s.audioStorage.Subir(
		context.Background(),
		objectKey,
		archivo.Contenido,
		archivo.Tamano,
		formatosAudioPermitidos[formato],
	); err != nil {
		tx.Rollback()
		return nil, ErrCancionErrorAlmacenamiento
	}

	codigoVersion, err :=
		s.cancionRepository.FinalizarCreacionVersionInicial(
			tx,
			codigoCancion,
			objectKey,
			formato,
		)

	if err != nil {
		return nil, err
	}

	return &dto.CrearCancionResponse{
		CodigoCancion:        codigoCancion,
		NombreCancion:        nombre,
		CodigoCancionVersion: codigoVersion,
		NumeroVersion:        1,
		EtiquetaVersion:      "v1.0.0",
	}, nil
}

func (s *cancionService) CrearVersion(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	archivo ArchivoAudio,
	notas *string,
	stems []ArchivoStem,
) (*dto.CrearVersionCancionResponse, error) {

	if archivo.Contenido == nil {
		return nil, ErrVersionPistaObligatoria
	}

	if archivo.Tamano > TamanoMaximoArchivoAudio {
		return nil, ErrCancionArchivoDemasiadoGrande
	}

	formato, err := formatoDesdeNombreArchivo(archivo.NombreOriginal)

	if err != nil {
		return nil, err
	}

	if notas != nil {
		notasLimpias := strings.TrimSpace(*notas)
		if notasLimpias == "" {
			notas = nil
		} else {
			notas = &notasLimpias
		}
	}

	formatosStems := make([]string, len(stems))

	for i, stem := range stems {

		if strings.TrimSpace(stem.Nombre) == "" {
			return nil, ErrStemNombreObligatorio
		}

		if stem.Tamano > TamanoMaximoArchivoAudio {
			return nil, ErrStemArchivoDemasiadoGrande
		}

		formatoStem, err := formatoDesdeNombreArchivo(stem.NombreOriginal)

		if err != nil {
			return nil, ErrStemFormatoInvalido
		}

		formatosStems[i] = formatoStem
	}

	existeCancion, err :=
		s.cancionRepository.ExisteCancionActivaEnProyecto(
			codigoProyecto,
			codigoCancion,
		)

	if err != nil {
		return nil, err
	}

	if !existeCancion {
		return nil, ErrVersionCancionNoEncontrada
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	puedeCrearVersion, err :=
		s.proyectoRepository.PuedeRealizarEnProyecto(
			integrante.CodIntegrante,
			codigoProyecto,
			"GESTIONAR_VERSIONES",
		)

	if err != nil {
		return nil, err
	}

	if !puedeCrearVersion {
		return nil, ErrVersionSinPermiso
	}

	tx, siguienteVersion, err :=
		s.cancionRepository.IniciarCreacionVersion(
			codigoCancion,
		)

	if err != nil {
		return nil, err
	}

	objectKey := claveObjetoAudio(codigoProyecto, codigoCancion, siguienteVersion, formato)

	if err := s.audioStorage.Subir(
		context.Background(),
		objectKey,
		archivo.Contenido,
		archivo.Tamano,
		formatosAudioPermitidos[formato],
	); err != nil {
		tx.Rollback()
		return nil, ErrCancionErrorAlmacenamiento
	}

	codigoVersion, err :=
		s.cancionRepository.InsertarVersion(
			tx,
			codigoCancion,
			siguienteVersion,
			objectKey,
			formato,
			notas,
		)

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	stemsCreados := make([]dto.StemResponse, 0, len(stems))

	for i, stem := range stems {

		stemObjectKey := claveObjetoStem(codigoProyecto, codigoCancion, siguienteVersion, i, formatosStems[i])

		if err := s.audioStorage.Subir(
			context.Background(),
			stemObjectKey,
			stem.Contenido,
			stem.Tamano,
			formatosAudioPermitidos[formatosStems[i]],
		); err != nil {
			tx.Rollback()
			return nil, ErrCancionErrorAlmacenamiento
		}

		codStem, err := s.cancionRepository.InsertarStem(
			tx,
			codigoVersion,
			strings.TrimSpace(stem.Nombre),
			stemObjectKey,
			formatosStems[i],
		)

		if err != nil {
			tx.Rollback()
			return nil, err
		}

		stemsCreados = append(stemsCreados, dto.StemResponse{
			CodStem: codStem,
			Nombre:  strings.TrimSpace(stem.Nombre),
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	etiqueta := "v1." +
		strconv.Itoa(siguienteVersion-1) +
		".0"

	return &dto.CrearVersionCancionResponse{
		CodigoCancionVersion: codigoVersion,
		CodigoCancion:        codigoCancion,
		NumeroVersion:        siguienteVersion,
		EtiquetaVersion:      etiqueta,
		Notas:                notas,
		Stems:                stemsCreados,
	}, nil
}

func (s *cancionService) ListarPorProyecto(
	codigoUsuario int64,
	codigoProyecto int64,
) ([]dto.CancionListadoResponse, error) {

	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrCancionProyectoNoEncontrado
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	esIntegrante, err :=
		s.proyectoRepository.EsIntegranteActivo(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !esIntegrante {
		return nil, ErrCancionSinAccesoProyecto
	}

	return s.cancionRepository.ListarPorProyecto(
		codigoProyecto,
	)
}

func (s *cancionService) ListarVersiones(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
) ([]dto.VersionCancionListadoResponse, error) {

	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrCancionProyectoNoEncontrado
	}

	existeCancion, err :=
		s.cancionRepository.ExisteCancionActivaEnProyecto(
			codigoProyecto,
			codigoCancion,
		)

	if err != nil {
		return nil, err
	}

	if !existeCancion {
		return nil, ErrVersionCancionNoEncontrada
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	esIntegrante, err :=
		s.proyectoRepository.EsIntegranteActivo(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !esIntegrante {
		return nil, ErrCancionSinAccesoProyecto
	}

	return s.cancionRepository.ListarVersiones(
		codigoCancion,
	)
}

func (s *cancionService) ObtenerURLDescargaVersion(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	codigoVersion int64,
) (*dto.AudioVersionResponse, error) {

	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrCancionProyectoNoEncontrado
	}

	existeCancion, err :=
		s.cancionRepository.ExisteCancionActivaEnProyecto(
			codigoProyecto,
			codigoCancion,
		)

	if err != nil {
		return nil, err
	}

	if !existeCancion {
		return nil, ErrVersionCancionNoEncontrada
	}

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	esIntegrante, err :=
		s.proyectoRepository.EsIntegranteActivo(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !esIntegrante {
		return nil, ErrCancionSinAccesoProyecto
	}

	version, err := s.cancionRepository.BuscarVersionPorCodigo(
		codigoCancion,
		codigoVersion,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrVersionCancionNoEncontrada
	}

	if err != nil {
		return nil, err
	}

	if version.URLArchivoCancionVer == nil || version.FormatoArchivoCancionVer == nil {
		return nil, ErrVersionSinArchivo
	}

	url, err := s.audioStorage.ObtenerURLDescarga(
		context.Background(),
		*version.URLArchivoCancionVer,
		VigenciaURLDescargaAudio,
	)

	if err != nil {
		log.Println("Error al generar URL de descarga de audio:", err)
		return nil, ErrCancionErrorAlmacenamiento
	}

	return &dto.AudioVersionResponse{
		CodigoCancionVersion: version.CodigoCancionVersion,
		URL:                  url,
		FormatoArchivo:       *version.FormatoArchivoCancionVer,
		ExpiraEnSegundos:     int(VigenciaURLDescargaAudio.Seconds()),
	}, nil
}

func (s *cancionService) ListarMisCanciones(
	codigoUsuario int64,
) ([]dto.MiCancionListadoResponse, error) {

	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	return s.cancionRepository.ListarPorIntegrante(
		integrante.CodIntegrante,
	)
}

func (s *cancionService) Editar(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
	request dto.EditarCancionRequest,
) (*dto.EditarCancionResponse, error) {

	nombre := strings.TrimSpace(
		request.Nombre,
	)

	if nombre == "" {
		return nil, ErrCancionNombreObligatorio
	}

	// Verificar que el proyecto siga activo.
	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !existeProyecto {
		return nil, ErrCancionProyectoNoEncontrado
	}

	// Verificar que la canción pertenezca a ese proyecto
	// y que no esté dada de baja.
	existeCancion, err :=
		s.cancionRepository.ExisteCancionActivaEnProyecto(
			codigoProyecto,
			codigoCancion,
		)

	if err != nil {
		return nil, err
	}

	if !existeCancion {
		return nil, ErrCancionNoEncontrada
	}

	// Obtener el perfil del usuario autenticado.
	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionPerfilRequerido
	}

	if err != nil {
		return nil, err
	}

	// La misma regla que para crear canciones:
	// debe tener GESTIONAR_CANCIONES en el proyecto.
	puedeGestionar, err :=
		s.proyectoRepository.PuedeGestionarCanciones(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return nil, err
	}

	if !puedeGestionar {
		return nil, ErrCancionSinPermiso
	}

	// Comprobar que no exista OTRA canción activa
	// con ese mismo nombre dentro del proyecto.
	existeNombre, err :=
		s.cancionRepository.
			ExisteNombreEnProyectoExceptoCancion(
				codigoProyecto,
				codigoCancion,
				nombre,
			)

	if err != nil {
		return nil, err
	}

	if existeNombre {
		return nil, ErrCancionNombreDuplicado
	}

	// Actualizar únicamente el nombre.
	err = s.cancionRepository.ActualizarNombre(
		codigoProyecto,
		codigoCancion,
		nombre,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCancionNoEncontrada
	}

	if err != nil {
		return nil, err
	}

	return &dto.EditarCancionResponse{
		CodigoCancion:  codigoCancion,
		CodigoProyecto: codigoProyecto,
		Nombre:         nombre,
	}, nil
}

func (s *cancionService) DarDeBaja(
	codigoUsuario int64,
	codigoProyecto int64,
	codigoCancion int64,
) error {

	// Verificar que el proyecto esté activo.
	existeProyecto, err :=
		s.proyectoRepository.ExisteProyectoActivo(
			codigoProyecto,
		)

	if err != nil {
		return err
	}

	if !existeProyecto {
		return ErrCancionProyectoNoEncontrado
	}

	// Verificar que la canción exista,
	// pertenezca al proyecto y esté activa.
	existeCancion, err :=
		s.cancionRepository.ExisteCancionActivaEnProyecto(
			codigoProyecto,
			codigoCancion,
		)

	if err != nil {
		return err
	}

	if !existeCancion {
		return ErrCancionNoEncontrada
	}

	// Obtener el integrante asociado al usuario.
	integrante, err :=
		s.integranteRepository.BuscarPorCodigoUsuario(
			codigoUsuario,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrCancionPerfilRequerido
	}

	if err != nil {
		return err
	}

	// Debe poder gestionar canciones en ese proyecto.
	puedeGestionar, err :=
		s.proyectoRepository.PuedeGestionarCanciones(
			integrante.CodIntegrante,
			codigoProyecto,
		)

	if err != nil {
		return err
	}

	if !puedeGestionar {
		return ErrCancionSinPermiso
	}

	// Baja lógica.
	err = s.cancionRepository.DarDeBaja(
		codigoProyecto,
		codigoCancion,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrCancionNoEncontrada
	}

	if err != nil {
		return err
	}

	return nil
}
