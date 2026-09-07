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
	ObtenerURLDescarga(ctx context.Context, objectKey string, vigencia time.Duration) (string, error)
}

type minioAudioStorage struct {
	client       *minio.Client
	publicClient *minio.Client
	bucket       string
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

	// El backend habla con MinIO por su nombre de servicio Docker
	// (MINIO_ENDPOINT, ej. "minio:9000"), pero las URLs presignadas las
	// resuelve el navegador del usuario, que no tiene ese nombre en su
	// propia red — necesita un host público (ej. "localhost:9000"). Se usa
	// un segundo cliente, con las mismas credenciales, solo para firmar.
	publicEndpoint := os.Getenv("MINIO_PUBLIC_ENDPOINT")

	if publicEndpoint == "" {
		publicEndpoint = endpoint
	}

	publicUseSSL := useSSL

	if valor, definido := os.LookupEnv("MINIO_PUBLIC_USE_SSL"); definido {
		publicUseSSL, _ = strconv.ParseBool(valor)
	}

	// Region fija: sin esto, PresignedGetObject dispara primero un
	// GetBucketLocation contra el propio endpoint del cliente para
	// resolverla, y ese endpoint público no es alcanzable desde dentro del
	// contenedor del backend (solo lo es desde el navegador).
	publicClient, err := minio.New(publicEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: publicUseSSL,
		Region: "us-east-1",
	})

	if err != nil {
		return nil, fmt.Errorf("error al preparar el cliente público de MinIO: %w", err)
	}

	return &minioAudioStorage{
		client:       client,
		publicClient: publicClient,
		bucket:       bucket,
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

func (s *minioAudioStorage) ObtenerURLDescarga(
	ctx context.Context,
	objectKey string,
	vigencia time.Duration,
) (string, error) {

	url, err := s.publicClient.PresignedGetObject(ctx, s.bucket, objectKey, vigencia, nil)

	if err != nil {
		return "", fmt.Errorf("error al generar la URL de descarga en MinIO: %w", err)
	}

	return url.String(), nil
}
