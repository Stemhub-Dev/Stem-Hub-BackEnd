package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type AudioStorage interface {
	Subir(ctx context.Context, objectKey string, contenido io.Reader, tamano int64, contentType string) error
}

type minioAudioStorage struct {
	client *minio.Client
	bucket string
}

func NewMinioAudioStorage() (AudioStorage, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucket := os.Getenv("MINIO_BUCKET")
	useSSL, _ := strconv.ParseBool(os.Getenv("MINIO_USE_SSL"))

	if bucket == "" {
		bucket = "stemhub-audio"
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		return nil, fmt.Errorf("error al preparar la conexión con MinIO: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	existe, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("error al verificar el bucket de MinIO: %w", err)
	}

	if !existe {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("error al crear el bucket de MinIO: %w", err)
		}
	}

	return &minioAudioStorage{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *minioAudioStorage) Subir(
	ctx context.Context,
	objectKey string,
	contenido io.Reader,
	tamano int64,
	contentType string,
) error {

	_, err := s.client.PutObject(
		ctx,
		s.bucket,
		objectKey,
		contenido,
		tamano,
		minio.PutObjectOptions{ContentType: contentType},
	)

	if err != nil {
		return fmt.Errorf("error al subir el archivo a MinIO: %w", err)
	}

	return nil
}
