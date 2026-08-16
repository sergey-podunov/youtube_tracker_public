package storage

import (
	"context"
	"io"

	"github.com/stretchr/testify/mock"
)

type MockThumbnailStorage struct {
	mock.Mock
}

func (m *MockThumbnailStorage) UploadThumbnail(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	args := m.Called(ctx, objectKey, reader, size, contentType)
	return args.Error(0)
}

type MockFileFetcher struct {
	mock.Mock
}

func (m *MockFileFetcher) FetchAndStore(ctx context.Context, fileURL string, objectKey string) error {
	args := m.Called(ctx, fileURL, objectKey)
	return args.Error(0)
}
