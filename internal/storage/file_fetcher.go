package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// FileFetcher downloads a file from a URL and stores it in object storage.
type FileFetcher interface {
	FetchAndStore(ctx context.Context, fileURL string, objectKey string) error
}

type FileFetcherImpl struct {
	storage    ThumbnailStorage
	httpClient *http.Client
	logger     *slog.Logger
}

const maxDownloadFileSize = int64(10 << 20) // 10MB

func NewFileFetcher(logger *slog.Logger, storage ThumbnailStorage, httpClient *http.Client) *FileFetcherImpl {
	return &FileFetcherImpl{
		storage:    storage,
		httpClient: httpClient,
		logger:     logger.With(slog.String("component", "FileFetcher")),
	}
}

func (fetcher *FileFetcherImpl) FetchAndStore(ctx context.Context, fileURL string, objectKey string) (err error) {
	if fileURL == "" {
		return nil
	}

	parsed, err := url.Parse(fileURL)
	if err != nil {
		return fmt.Errorf("invalid URL %s: %w", fileURL, err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", parsed.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return fmt.Errorf("create request for %s: %w", fileURL, err)
	}

	resp, err := fetcher.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", fileURL, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %d", fileURL, resp.StatusCode)
	}

	if resp.ContentLength > maxDownloadFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", resp.ContentLength, maxDownloadFileSize)
	}
	reader := io.LimitReader(resp.Body, maxDownloadFileSize)

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	if err := fetcher.storage.UploadThumbnail(ctx, objectKey, reader, resp.ContentLength, contentType); err != nil {
		return fmt.Errorf("store %s: %w", objectKey, err)
	}

	fetcher.logger.Info("Fetched and stored file", slog.String("url", fileURL), slog.String("object_key", objectKey))
	return nil
}
