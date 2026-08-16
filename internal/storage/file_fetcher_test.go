package storage_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"youtube_tracker/internal/storage"
	"youtube_tracker/internal/youtube/test_utils"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestFileFetcherTestSuite(t *testing.T) {
	suite.Run(t, new(FileFetcherTestSuite))
}

type FileFetcherTestSuite struct {
	suite.Suite
	mockStorage *storage.MockThumbnailStorage
	fetcher     *storage.FileFetcherImpl
}

func (suite *FileFetcherTestSuite) SetupTest() {
	suite.mockStorage = new(storage.MockThumbnailStorage)
	suite.fetcher = storage.NewFileFetcher(test_utils.NewNopLogger(), suite.mockStorage, http.DefaultClient)
}

func (suite *FileFetcherTestSuite) TestFetchAndStore_Success() {
	ctx := suite.T().Context()
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG magic bytes

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer server.Close()

	suite.mockStorage.On("UploadThumbnail", ctx, "UC123abc.jpg", mock.Anything, mock.Anything, "image/jpeg").Return(nil)

	err := suite.fetcher.FetchAndStore(ctx, server.URL+"/thumb.jpg", "UC123abc.jpg")
	suite.NoError(err)

	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *FileFetcherTestSuite) TestFetchAndStore_DownloadError() {
	ctx := suite.T().Context()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := suite.fetcher.FetchAndStore(ctx, server.URL+"/thumb.jpg", "UC123abc.jpg")
	suite.Error(err)

	suite.mockStorage.AssertNotCalled(suite.T(), "UploadThumbnail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (suite *FileFetcherTestSuite) TestFetchAndStore_UploadError() {
	ctx := suite.T().Context()
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer server.Close()

	suite.mockStorage.On("UploadThumbnail", ctx, "UC123abc.jpg", mock.Anything, mock.Anything, "image/jpeg").Return(fmt.Errorf("s3 error"))

	err := suite.fetcher.FetchAndStore(ctx, server.URL+"/thumb.jpg", "UC123abc.jpg")
	suite.Error(err)

	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *FileFetcherTestSuite) TestFetchAndStore_EmptyURL() {
	ctx := suite.T().Context()

	err := suite.fetcher.FetchAndStore(ctx, "", "UC123abc.jpg")
	suite.NoError(err)

	suite.mockStorage.AssertNotCalled(suite.T(), "UploadThumbnail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (suite *FileFetcherTestSuite) TestFetchAndStore_RejectsUnsupportedScheme() {
	ctx := suite.T().Context()

	err := suite.fetcher.FetchAndStore(ctx, "ftp://example.com/thumb.jpg", "UC123abc.jpg")
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported URL scheme")

	suite.mockStorage.AssertNotCalled(suite.T(), "UploadThumbnail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
