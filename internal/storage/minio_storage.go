package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

type FileStorage interface {
	DescargarArchivo(ctx context.Context, bucket, objectName string) ([]byte, error)
}

func (s *minioAudioStorage) DescargarArchivo(
	ctx context.Context,
	bucket string,
	objectName string,
) ([]byte, error) {

	object, err := s.client.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("error al obtener el archivo desde MinIO: %w", err)
	}
	defer object.Close()

	if _, err := object.Stat(); err != nil {
		return nil, fmt.Errorf("error al verificar el archivo en MinIO: %w", err)
	}

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("error al leer el contenido del archivo desde MinIO: %w", err)
	}

	return data, nil
}
