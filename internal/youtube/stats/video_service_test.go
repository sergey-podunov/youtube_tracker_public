//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestYoutubeVideoService_CreateVideo(t *testing.T) {
	ctx := t.Context()

	type testCase struct {
		name string
		test func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository)
	}

	tests := []testCase{
		{
			"getChannel error", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{}, false, errors.New("db error"))

				input := YoutubeVideo{YoutubeChannelId: 42, ExternalID: "vid_ext_1"}
				actual, created, err := service.CreateVideo(ctx, input)
				assert.Error(t, err)
				assert.False(t, created)
				assert.Equal(t, YoutubeVideo{}, actual)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertNotCalled(t, "getVideoByExternalId", mock.Anything, mock.Anything, mock.Anything)
				videoRepo.AssertNotCalled(t, "createVideo", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			"channel not found", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{}, false, nil)

				input := YoutubeVideo{YoutubeChannelId: 42, ExternalID: "vid_ext_1"}
				actual, created, err := service.CreateVideo(ctx, input)
				assert.NoError(t, err)
				assert.False(t, created)
				assert.Equal(t, YoutubeVideo{}, actual)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertNotCalled(t, "getVideoByExternalId", mock.Anything, mock.Anything, mock.Anything)
				videoRepo.AssertNotCalled(t, "createVideo", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			"video already exists", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{YoutubeChannelId: 42}, true, nil)

				existing := YoutubeVideo{
					YoutubeVideoId:   100,
					YoutubeChannelId: 42,
					ExternalID:       "vid_ext_1",
					Title:            "Existing",
				}
				videoRepo.On("getVideoByExternalId", ctx, mock.Anything, "vid_ext_1").
					Return(existing, true, nil)

				input := YoutubeVideo{YoutubeChannelId: 42, ExternalID: "vid_ext_1", Title: "Existing"}
				actual, created, err := service.CreateVideo(ctx, input)
				assert.NoError(t, err)
				assert.False(t, created)
				assert.Equal(t, existing, actual)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertExpectations(t)
				videoRepo.AssertNotCalled(t, "createVideo", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			"getVideoByExternalId error", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{YoutubeChannelId: 42}, true, nil)
				videoRepo.On("getVideoByExternalId", ctx, mock.Anything, "vid_ext_1").
					Return(YoutubeVideo{}, false, errors.New("db error"))

				input := YoutubeVideo{YoutubeChannelId: 42, ExternalID: "vid_ext_1"}
				actual, created, err := service.CreateVideo(ctx, input)
				assert.Error(t, err)
				assert.False(t, created)
				assert.Equal(t, YoutubeVideo{}, actual)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertExpectations(t)
				videoRepo.AssertNotCalled(t, "createVideo", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			"new video created", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{YoutubeChannelId: 42}, true, nil)
				videoRepo.On("getVideoByExternalId", ctx, mock.Anything, "vid_ext_1").
					Return(YoutubeVideo{}, false, nil)

				input := YoutubeVideo{YoutubeChannelId: 42, ExternalID: "vid_ext_1", Title: "New"}
				created := YoutubeVideo{YoutubeVideoId: 200, YoutubeChannelId: 42, ExternalID: "vid_ext_1", Title: "New", CreatedAt: time.Now()}
				videoRepo.On("createVideo", ctx, mock.Anything, input).Return(created, nil)

				actual, isNew, err := service.CreateVideo(ctx, input)
				require.NoError(t, err)
				assert.True(t, isNew)
				assert.Equal(t, created, actual)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertExpectations(t)
			},
		},
		{
			"createVideo error", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{YoutubeChannelId: 42}, true, nil)
				videoRepo.On("getVideoByExternalId", ctx, mock.Anything, "vid_ext_1").
					Return(YoutubeVideo{}, false, nil)

				input := YoutubeVideo{YoutubeChannelId: 42, ExternalID: "vid_ext_1"}
				videoRepo.On("createVideo", ctx, mock.Anything, input).
					Return(YoutubeVideo{}, errors.New("insert error"))

				actual, created, err := service.CreateVideo(ctx, input)
				assert.Error(t, err)
				assert.False(t, created)
				assert.Equal(t, YoutubeVideo{}, actual)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertExpectations(t)
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockTxController := new(MockTxController)
			mockTxController.On("Begin", ctx).Return(&MockTx{}, nil)

			videoRepo := new(internalMockVideoRepository)
			channelRepo := new(internalMockChannelRepository)
			service := NewYoutubeVideoService(logger, mockTxController, videoRepo, channelRepo)

			tc.test(t, service, videoRepo, channelRepo)
		})
	}
}

func TestYoutubeVideoService_GetVideoStats(t *testing.T) {
	ctx := t.Context()

	type testCase struct {
		name string
		test func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository)
	}

	tests := []testCase{
		{
			"getVideo error", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository) {
				videoRepo.On("getVideo", ctx, mock.Anything, int64(100)).
					Return(YoutubeVideo{}, false, errors.New("db error"))

				info, ok, err := service.GetVideoStats(ctx, 100)
				assert.Error(t, err)
				assert.False(t, ok)
				assert.Equal(t, YoutubeVideoStatsInfo{}, info)

				videoRepo.AssertExpectations(t)
				videoRepo.AssertNotCalled(t, "getVideoStat", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			"video not found", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository) {
				videoRepo.On("getVideo", ctx, mock.Anything, int64(100)).
					Return(YoutubeVideo{}, false, nil)

				info, ok, err := service.GetVideoStats(ctx, 100)
				assert.NoError(t, err)
				assert.False(t, ok)
				assert.Equal(t, YoutubeVideoStatsInfo{}, info)

				videoRepo.AssertExpectations(t)
				videoRepo.AssertNotCalled(t, "getVideoStat", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			"getVideoStat error", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository) {
				videoRepo.On("getVideo", ctx, mock.Anything, int64(100)).
					Return(YoutubeVideo{YoutubeVideoId: 100, ExternalID: "vid_ext_1"}, true, nil)
				videoRepo.On("getVideoStat", ctx, mock.Anything, int64(100)).
					Return(nil, errors.New("db error"))

				info, ok, err := service.GetVideoStats(ctx, 100)
				assert.Error(t, err)
				assert.False(t, ok)
				assert.Equal(t, YoutubeVideoStatsInfo{}, info)

				videoRepo.AssertExpectations(t)
			},
		},
		{
			"empty stats", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository) {
				videoRepo.On("getVideo", ctx, mock.Anything, int64(100)).
					Return(YoutubeVideo{YoutubeVideoId: 100, ExternalID: "vid_ext_1"}, true, nil)
				videoRepo.On("getVideoStat", ctx, mock.Anything, int64(100)).
					Return([]YoutubeVideoStats{}, nil)

				info, ok, err := service.GetVideoStats(ctx, 100)
				require.NoError(t, err)
				assert.True(t, ok)
				assert.Equal(t, int64(100), info.ID)
				assert.Equal(t, "vid_ext_1", info.VideoID)
				assert.Empty(t, info.Statistics)

				videoRepo.AssertExpectations(t)
			},
		},
		{
			"stats exist truncated to day", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository) {
				videoRepo.On("getVideo", ctx, mock.Anything, int64(100)).
					Return(YoutubeVideo{YoutubeVideoId: 100, ExternalID: "vid_ext_1"}, true, nil)
				videoRepo.On("getVideoStat", ctx, mock.Anything, int64(100)).
					Return([]YoutubeVideoStats{
						{
							YoutubeVideoStatID: 1,
							YoutubeVideoID:     100,
							ViewCount:          500,
							LikeCount:          50,
							CommentCount:       5,
							CreatedAt:          helpers.ParseTime("2025-11-12T05:06:07Z"),
						},
					}, nil)

				info, ok, err := service.GetVideoStats(ctx, 100)
				require.NoError(t, err)
				assert.True(t, ok)
				assert.Equal(t, int64(100), info.ID)
				assert.Equal(t, "vid_ext_1", info.VideoID)
				require.Len(t, info.Statistics, 1)

				stat := info.Statistics[0]
				assert.Equal(t, int64(500), stat.ViewCount)
				assert.Equal(t, int64(50), stat.LikeCount)
				assert.Equal(t, int64(5), stat.CommentCount)
				assert.Equal(t, helpers.ParseTime("2025-11-12T00:00:00Z"), stat.CreatedAt)

				videoRepo.AssertExpectations(t)
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockTxController := new(MockTxController)
			mockTxController.On("Begin", ctx).Return(&MockTx{}, nil)

			videoRepo := new(internalMockVideoRepository)
			channelRepo := new(internalMockChannelRepository)
			service := NewYoutubeVideoService(logger, mockTxController, videoRepo, channelRepo)

			tc.test(t, service, videoRepo)
		})
	}
}

func TestYoutubeVideoService_GetVideosByChannel(t *testing.T) {
	ctx := t.Context()

	type testCase struct {
		name string
		test func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository)
	}

	tests := []testCase{
		{
			"getChannel error", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{}, false, errors.New("db error"))

				info, found, err := service.GetVideosByChannel(ctx, 42, 1, 10)
				assert.Error(t, err)
				assert.False(t, found)
				assert.Equal(t, YoutubeVideosInfo{}, info)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertNotCalled(t, "getVideosByChannelPaginated", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				videoRepo.AssertNotCalled(t, "getVideosByChannelCount", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			"channel not found", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{}, false, nil)

				info, found, err := service.GetVideosByChannel(ctx, 42, 1, 10)
				assert.NoError(t, err)
				assert.False(t, found)
				assert.Equal(t, YoutubeVideosInfo{}, info)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertNotCalled(t, "getVideosByChannelPaginated", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				videoRepo.AssertNotCalled(t, "getVideosByChannelCount", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			"empty db", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{YoutubeChannelId: 42}, true, nil)
				videoRepo.On("getVideosByChannelPaginated", ctx, mock.Anything, int64(42), 0, 20).Return([]YoutubeVideo{}, nil)
				videoRepo.On("getVideosByChannelCount", ctx, mock.Anything, int64(42)).Return(0, nil)

				info, found, err := service.GetVideosByChannel(ctx, 42, 1, 20)
				require.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, 1, info.CurrentPage)
				assert.Equal(t, 20, info.PageSize)
				assert.Equal(t, 0, info.TotalPages)
				assert.Empty(t, info.Videos)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertExpectations(t)
			},
		},
		{
			"full db second page", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{YoutubeChannelId: 42}, true, nil)

				videos := []YoutubeVideo{
					{YoutubeVideoId: 200, YoutubeChannelId: 42, ExternalID: "vid_ext_1", Title: "v1"},
				}
				videoRepo.On("getVideosByChannelPaginated", ctx, mock.Anything, int64(42), 5, 5).Return(videos, nil)
				videoRepo.On("getVideosByChannelCount", ctx, mock.Anything, int64(42)).Return(6, nil)

				info, found, err := service.GetVideosByChannel(ctx, 42, 2, 5)
				require.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, 2, info.CurrentPage)
				assert.Equal(t, 5, info.PageSize)
				assert.Equal(t, 2, info.TotalPages)
				require.Len(t, info.Videos, 1)
				assert.Equal(t, int64(200), info.Videos[0].YoutubeVideoId)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertExpectations(t)
			},
		},
		{
			"without params uses defaults", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{YoutubeChannelId: 42}, true, nil)

				videos := []YoutubeVideo{{YoutubeVideoId: 200, YoutubeChannelId: 42, ExternalID: "vid_ext_1"}}
				videoRepo.On("getVideosByChannelPaginated", ctx, mock.Anything, int64(42), 0, 20).Return(videos, nil)
				videoRepo.On("getVideosByChannelCount", ctx, mock.Anything, int64(42)).Return(6, nil)

				info, found, err := service.GetVideosByChannel(ctx, 42, 0, 0)
				require.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, 1, info.CurrentPage)
				assert.Equal(t, 20, info.PageSize)
				assert.Equal(t, 1, info.TotalPages)
				require.Len(t, info.Videos, 1)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertExpectations(t)
			},
		},
		{
			"page size too big is capped", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{YoutubeChannelId: 42}, true, nil)

				videos := []YoutubeVideo{{YoutubeVideoId: 200, YoutubeChannelId: 42, ExternalID: "vid_ext_1"}}
				videoRepo.On("getVideosByChannelPaginated", ctx, mock.Anything, int64(42), 0, 20).Return(videos, nil)
				videoRepo.On("getVideosByChannelCount", ctx, mock.Anything, int64(42)).Return(6, nil)

				info, found, err := service.GetVideosByChannel(ctx, 42, 1, DefaultPageSize*2)
				require.NoError(t, err)
				assert.True(t, found)
				assert.Equal(t, 1, info.CurrentPage)
				assert.Equal(t, 20, info.PageSize)
				assert.Equal(t, 1, info.TotalPages)
				require.Len(t, info.Videos, 1)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertExpectations(t)
			},
		},
		{
			"getVideosByChannelPaginated error", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{YoutubeChannelId: 42}, true, nil)
				videoRepo.On("getVideosByChannelPaginated", ctx, mock.Anything, int64(42), 0, 20).
					Return(nil, errors.New("db error"))

				_, found, err := service.GetVideosByChannel(ctx, 42, 1, 20)
				assert.Error(t, err)
				assert.True(t, found)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertExpectations(t)
				videoRepo.AssertNotCalled(t, "getVideosByChannelCount", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			"getVideosByChannelCount error", func(t *testing.T, service VideoService, videoRepo *internalMockVideoRepository, channelRepo *internalMockChannelRepository) {
				channelRepo.On("getChannel", ctx, mock.Anything, int64(42)).
					Return(YoutubeChannel{YoutubeChannelId: 42}, true, nil)
				videoRepo.On("getVideosByChannelPaginated", ctx, mock.Anything, int64(42), 0, 20).
					Return([]YoutubeVideo{}, nil)
				videoRepo.On("getVideosByChannelCount", ctx, mock.Anything, int64(42)).
					Return(0, errors.New("db error"))

				_, found, err := service.GetVideosByChannel(ctx, 42, 1, 20)
				assert.Error(t, err)
				assert.True(t, found)

				channelRepo.AssertExpectations(t)
				videoRepo.AssertExpectations(t)
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockTxController := new(MockTxController)
			mockTxController.On("Begin", ctx).Return(&MockTx{}, nil)

			videoRepo := new(internalMockVideoRepository)
			channelRepo := new(internalMockChannelRepository)
			service := NewYoutubeVideoService(logger, mockTxController, videoRepo, channelRepo)

			tc.test(t, service, videoRepo, channelRepo)
		})
	}
}
