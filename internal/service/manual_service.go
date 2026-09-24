package service

import (
	"context"
	"errors"

	"github.com/facu-1538/Stem-Hub-BackEnd/internal/storage"
)

var (
	ErrManualNoEncontrado = errors.New("manual de usuario no encontrado")
)

type ManualService struct {
	storage    storage.FileStorage
	bucket     string
	objectName string
}

func NewManualService(storage storage.FileStorage, bucket, objectName string) *ManualService {
	if bucket == "" {
		bucket = "documentos"
	}
	if objectName == "" {
		objectName = "manual-usuario.pdf"
	}

	return &ManualService{
		storage:    storage,
		bucket:     bucket,
		objectName: objectName,
	}
}

func (s *ManualService) ObtenerManualUsuario(ctx context.Context) ([]byte, error) {
	contenido, err := s.storage.DescargarArchivo(ctx, s.bucket, s.objectName)
	if err != nil {
		return nil, ErrManualNoEncontrado
	}

	return contenido, nil
}
