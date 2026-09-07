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
		s.cancionRepository.FinalizarCreacionVersion(
			tx,
			codigoCancion,
			siguienteVersion,
			objectKey,
			formato,
		)

	if err != nil {
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
