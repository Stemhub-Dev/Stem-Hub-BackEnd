package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/model"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/repository"
	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
)

const TamanoMaximoAvatar int64 = 5 * 1024 * 1024 // 5 MB

var formatosImagenPermitidos = map[string]string{
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"webp": "image/webp",
}

// VigenciaURLAvatar es el tiempo de validez de la URL presignada devuelta
// para mostrar el avatar del perfil.
const VigenciaURLAvatar = 1 * time.Hour

var (
	ErrPerfilNoEncontrado = errors.New(
		"el usuario no posee un perfil",
	)

	ErrPerfilNombreObligatorio = errors.New(
		"el nombre no puede estar vacío",
	)

	ErrPerfilNombreLargo = errors.New(
		"el nombre es demasiado largo",
	)

	ErrPerfilAvatarFormatoInvalido = errors.New(
		"el formato de la imagen no está soportado",
	)

	ErrPerfilAvatarDemasiadoGrande = errors.New(
		"la imagen supera el tamaño máximo permitido",
	)

	ErrPerfilErrorAlmacenamiento = errors.New(
		"error al almacenar el avatar",
	)
)

// ArchivoImagen representa el archivo de avatar recibido por el handler,
// desacoplado de multipart.FileHeader para no filtrar detalles de Gin al
// service.
type ArchivoImagen struct {
	Contenido      io.Reader
	NombreOriginal string
	Tamano         int64
}

func formatoImagenDesdeNombreArchivo(nombreArchivo string) (string, error) {
	partes := strings.Split(nombreArchivo, ".")

	if len(partes) < 2 {
		return "", ErrPerfilAvatarFormatoInvalido
	}

	extension := strings.ToLower(partes[len(partes)-1])

	if _, ok := formatosImagenPermitidos[extension]; !ok {
		return "", ErrPerfilAvatarFormatoInvalido
	}

	return extension, nil
}

func claveObjetoAvatar(codigoIntegrante int64, formato string) string {
	return fmt.Sprintf(
		"integrantes/%d/avatar.%s",
		codigoIntegrante,
		formato,
	)
}

type IntegranteService interface {
	ObtenerPerfil(
		codigoUsuario int64,
	) (*model.Integrante, *string, error)

	EditarPerfil(
		codigoUsuario int64,
		nombre string,
		descripcion *string,
		avatar *ArchivoImagen,
		eliminarAvatar bool,
	) (*model.Integrante, *string, error)
}

type integranteService struct {
	repository   repository.IntegranteRepository
	audioStorage storage.AudioStorage
}

func NewIntegranteService(
	repository repository.IntegranteRepository,
	audioStorage storage.AudioStorage,
) IntegranteService {
	return &integranteService{
		repository:   repository,
		audioStorage: audioStorage,
	}
}

func (s *integranteService) resolverAvatarUrl(
	integrante *model.Integrante,
) (*string, error) {

	if integrante.AvatarObjectKey == nil {
		return nil, nil
	}

	url, err := s.audioStorage.ObtenerURLDescarga(
		context.Background(),
		*integrante.AvatarObjectKey,
		VigenciaURLAvatar,
	)

	if err != nil {
		return nil, err
	}

	return &url, nil
}

// Best-effort: si falla queda un objeto huérfano en MinIO, pero el perfil
// ya no lo referencia y no tiene sentido revertir la operación del usuario.
func (s *integranteService) eliminarArchivoAvatar(objectKey string) {
	if err := s.audioStorage.Eliminar(context.Background(), objectKey); err != nil {
		log.Println("No se pudo eliminar el avatar en MinIO:", objectKey, err)
	}
}

func (s *integranteService) buscarPerfilActivo(
	codigoUsuario int64,
) (*model.Integrante, error) {

	integrante, err :=
		s.repository.BuscarPorCodigoUsuario(codigoUsuario)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPerfilNoEncontrado
	}

	if err != nil {
		return nil, err
	}

	if integrante.FechaHoraBajaIntegrante != nil {
		return nil, ErrPerfilNoEncontrado
	}

	return integrante, nil
}

func (s *integranteService) ObtenerPerfil(
	codigoUsuario int64,
) (*model.Integrante, *string, error) {

	integrante, err := s.buscarPerfilActivo(codigoUsuario)

	if err != nil {
		return nil, nil, err
	}

	avatarUrl, err := s.resolverAvatarUrl(integrante)

	if err != nil {
		return nil, nil, err
	}

	return integrante, avatarUrl, nil
}

func (s *integranteService) EditarPerfil(
	codigoUsuario int64,
	nombre string,
	descripcion *string,
	avatar *ArchivoImagen,
	eliminarAvatar bool,
) (*model.Integrante, *string, error) {

	nombre = strings.TrimSpace(nombre)

	if nombre == "" {
		return nil, nil, ErrPerfilNombreObligatorio
	}

	if utf8.RuneCountInString(nombre) > LargoMaximoNombreIntegrante {
		return nil, nil, ErrPerfilNombreLargo
	}

	integrante, err := s.buscarPerfilActivo(codigoUsuario)

	if err != nil {
		return nil, nil, err
	}

	avatarObjectKey := integrante.AvatarObjectKey
	avatarAnterior := integrante.AvatarObjectKey

	if avatar != nil && avatar.Contenido != nil {

		if avatar.Tamano > TamanoMaximoAvatar {
			return nil, nil, ErrPerfilAvatarDemasiadoGrande
		}

		formato, err := formatoImagenDesdeNombreArchivo(avatar.NombreOriginal)

		if err != nil {
			return nil, nil, err
		}

		objectKey := claveObjetoAvatar(integrante.CodIntegrante, formato)

		if err := s.audioStorage.Subir(
			context.Background(),
			objectKey,
			avatar.Contenido,
			avatar.Tamano,
			formatosImagenPermitidos[formato],
		); err != nil {
			return nil, nil, ErrPerfilErrorAlmacenamiento
		}

		avatarObjectKey = &objectKey
	} else if eliminarAvatar {
		// Un avatar nuevo tiene prioridad sobre el pedido de quitarlo.
		avatarObjectKey = nil
	}

	if err := s.repository.ActualizarPerfil(
		integrante.CodIntegrante,
		nombre,
		descripcion,
		avatarObjectKey,
	); err != nil {
		return nil, nil, err
	}

	// Si se quitó el avatar, o si el formato cambió (p. ej. png -> jpg) y la
	// clave nueva es distinta, el archivo anterior quedaría huérfano.
	if avatarAnterior != nil &&
		(avatarObjectKey == nil || *avatarAnterior != *avatarObjectKey) {
		s.eliminarArchivoAvatar(*avatarAnterior)
	}

	integrante.NombreIntegrante = nombre
	integrante.DescripcionIntegrante = descripcion
	integrante.AvatarObjectKey = avatarObjectKey

	avatarUrl, err := s.resolverAvatarUrl(integrante)

	if err != nil {
		return nil, nil, err
	}

	return integrante, avatarUrl, nil
}
