package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/minio/minio-go/v7"
)

// ThumbnailStorage stores thumbnail images in object storage.
type ThumbnailStorage interface {
	UploadThumbnail(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error
}

type S3StorageMinio struct {
	client *minio.Client
	bucket string
	logger *slog.Logger
}

func NewS3StorageMinio(logger *slog.Logger, client *minio.Client, bucket string) *S3StorageMinio {
	return &S3StorageMinio{
		client: client,
		bucket: bucket,
		logger: logger.With(slog.String("component", "S3ThumbnailStorage")),
	}
}

func (storage *S3StorageMinio) UploadThumbnail(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	_, err := storage.client.PutObject(ctx, storage.bucket, objectKey, reader, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		storage.logger.Error("Failed to upload object", slog.String("object_key", objectKey), "error", err)
		return fmt.Errorf("upload %s: %w", objectKey, err)
	}
	storage.logger.Info("Object uploaded", slog.String("object_key", objectKey))
	return nil
}
