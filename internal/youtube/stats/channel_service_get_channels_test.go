//go:build database

//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestYoutubeChannelService_GetChannels(t *testing.T) {
	ctx := t.Context()

	type testCase struct {
		name string
		test func(t *testing.T, service *YoutubeChannelService, repository *internalMockChannelRepository)
	}

	tests := []testCase{
		{
			"empty db", func(t *testing.T, service *YoutubeChannelService, repository *internalMockChannelRepository) {
				repository.On("getChannelsPaginated", ctx, mock.Anything, 0, 20).Return([]YoutubeChannel{}, nil)
				repository.On("getChannelsCount", ctx, mock.Anything).Return(0, nil)

				actualChannelsInfo, err := service.GetChannels(ctx, 1, 20)
				require.NoError(t, err)

				assert.Equal(t, 1, actualChannelsInfo.CurrentPage)
				assert.Equal(t, 20, actualChannelsInfo.PageSize)
				assert.Equal(t, 0, actualChannelsInfo.TotalPages)
				assert.Equal(t, 0, len(actualChannelsInfo.Channels))

				repository.AssertExpectations(t)
			},
		},
		{
			"full db", func(t *testing.T, service *YoutubeChannelService, repository *internalMockChannelRepository) {
				youtubeChannels := []YoutubeChannel{
					{
						YoutubeChannelId: int64(345),
						ExternalID:       "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
						Title:            "Google Developers",
						CreatedAt:        time.Now(),
					},
				}
				repository.On("getChannelsPaginated", ctx, mock.Anything, 5, 5).Return(youtubeChannels, nil)
				repository.On("getChannelsCount", ctx, mock.Anything).Return(6, nil)

				actualChannelsInfo, err := service.GetChannels(ctx, 2, 5)
				require.NoError(t, err)

				assert.Equal(t, 2, actualChannelsInfo.CurrentPage)
				assert.Equal(t, 5, actualChannelsInfo.PageSize)
				assert.Equal(t, 2, actualChannelsInfo.TotalPages)
				assert.Equal(t, 1, len(actualChannelsInfo.Channels))

				actualChannel := actualChannelsInfo.Channels[0]
				assert.Equal(t, int64(345), actualChannel.YoutubeChannelId)

				repository.AssertExpectations(t)
			},
		},
		{
			"without params", func(t *testing.T, service *YoutubeChannelService, repository *internalMockChannelRepository) {
				youtubeChannels := []YoutubeChannel{
					{
						YoutubeChannelId: int64(345),
						ExternalID:       "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
						Title:            "Google Developers",
						CreatedAt:        time.Now(),
					},
				}
				repository.On("getChannelsPaginated", ctx, mock.Anything, 0, 20).Return(youtubeChannels, nil)
				repository.On("getChannelsCount", ctx, mock.Anything).Return(6, nil)

				actualChannelsInfo, err := service.GetChannels(ctx, 0, 0)
				require.NoError(t, err)

				assert.Equal(t, 1, actualChannelsInfo.CurrentPage)
				assert.Equal(t, 20, actualChannelsInfo.PageSize)
				assert.Equal(t, 1, actualChannelsInfo.TotalPages)
				assert.Equal(t, 1, len(actualChannelsInfo.Channels))

				actualChannel := actualChannelsInfo.Channels[0]
				assert.Equal(t, int64(345), actualChannel.YoutubeChannelId)

				repository.AssertExpectations(t)
			},
		},
		{
			"page size is too big", func(t *testing.T, service *YoutubeChannelService, repository *internalMockChannelRepository) {
				youtubeChannels := []YoutubeChannel{
					{
						YoutubeChannelId: int64(345),
						ExternalID:       "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
						Title:            "Google Developers",
						CreatedAt:        time.Now(),
					},
				}
				repository.On("getChannelsPaginated", ctx, mock.Anything, 0, 20).Return(youtubeChannels, nil)
				repository.On("getChannelsCount", ctx, mock.Anything).Return(6, nil)

				actualChannelsInfo, err := service.GetChannels(ctx, 1, DefaultPageSize*2)
				require.NoError(t, err)

				assert.Equal(t, 1, actualChannelsInfo.CurrentPage)
				assert.Equal(t, 20, actualChannelsInfo.PageSize)
				assert.Equal(t, 1, actualChannelsInfo.TotalPages)
				assert.Equal(t, 1, len(actualChannelsInfo.Channels))

				actualChannel := actualChannelsInfo.Channels[0]
				assert.Equal(t, int64(345), actualChannel.YoutubeChannelId)

				repository.AssertExpectations(t)
			},
		},
	}

	for _, tc := range tests {

		mockTxtController := new(MockTxController)
		mockTxtController.On("Begin", ctx).Return(&MockTx{}, nil)

		mockInternalRepository := new(internalMockChannelRepository)

		service := NewYoutubeChannelService(slog.New(slog.NewTextHandler(io.Discard, nil)), mockTxtController, mockInternalRepository, nil, nil)

		t.Run(tc.name, func(t *testing.T) {
			tc.test(t, service, mockInternalRepository)
		})
	}
}

func TestYoutubeChannelService_getOffsetByPage(t *testing.T) {
	type testCase struct {
		name           string
		page           int
		pageSize       int
		expectedOffset int
	}

	tests := []testCase{
		{"empty page", 0, 10, 0},
		{"first page", 1, 10, 0},
		{"next page", 2, 10, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualOffset := getOffsetByPage(tc.page, tc.pageSize)
			assert.Equal(t, tc.expectedOffset, actualOffset)
		})
	}
}

func TestYoutubeChannelService_getTotalPages(t *testing.T) {
	type testCase struct {
		name               string
		count              int
		pageSize           int
		expectedPagesCount int
	}

	tests := []testCase{
		{"no channels", 0, 10, 0},
		{"in limit for the first page", 5, 10, 1},
		{"max for the first page", 10, 10, 1},
		{"in limit for the next page", 11, 10, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualPageCount := getTotalPages(tc.count, tc.pageSize)
			assert.Equal(t, tc.expectedPagesCount, actualPageCount)
		})
	}
}
