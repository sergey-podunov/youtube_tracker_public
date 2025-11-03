//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"errors"
	"testing"
	"youtube_tracker/internal/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestYoutubeChannelService_GetChannelStat(t *testing.T) {
	ctx := t.Context()

	type testCase struct {
		name string
		test func(t *testing.T, mockInternalRepository *internalMockChannelRepository, service ChannelService)
	}

	tests := []testCase{
		{
			"error", func(t *testing.T, mockInternalRepository *internalMockChannelRepository, service ChannelService) {
				mockInternalRepository.On("getChannel", ctx, mock.Anything, int64(84374)).Return(nil, false, errors.New("some error"))

				channelStats, ok, err := service.GetChannelStats(ctx, 84374, nil, nil)
				assert.Error(t, err)
				assert.Equal(t, YoutubeChannelStatsInfo{}, channelStats)
				assert.False(t, ok)

				mockInternalRepository.AssertExpectations(t)
			},
		},
		{
			"no channel", func(t *testing.T, mockInternalRepository *internalMockChannelRepository, service ChannelService) {
				mockInternalRepository.On("getChannel", ctx, mock.Anything, int64(84374)).Return(nil, false, nil)

				channelStats, ok, err := service.GetChannelStats(ctx, 84374, nil, nil)
				assert.NoError(t, err)
				assert.Equal(t, YoutubeChannelStatsInfo{}, channelStats)
				assert.False(t, ok)

				mockInternalRepository.AssertExpectations(t)
			},
		},
		{
			"empty db", func(t *testing.T, mockInternalRepository *internalMockChannelRepository, service ChannelService) {
				mockInternalRepository.On("getChannel", ctx, mock.Anything, int64(84374)).Return(&YoutubeChannel{
					YoutubeChannelId: int64(84374),
					ExternalID:       "3263yw",
					Name:             "Some Channel",
				}, true, nil)
				mockInternalRepository.On("getChannelStat", ctx, mock.Anything, int64(84374)).Return([]YoutubeChannelStats{}, nil)

				channelStats, ok, err := service.GetChannelStats(ctx, 84374, nil, nil)
				require.NoError(t, err)
				require.NotNil(t, channelStats)
				assert.True(t, ok)
				assert.Equal(t, int64(84374), channelStats.ID)
				assert.Equal(t, "3263yw", channelStats.ChannelID)
				assert.Equal(t, 0, len(channelStats.Statistics))

				mockInternalRepository.AssertExpectations(t)
			},
		},
		{
			"statistics exists", func(t *testing.T, mockInternalRepository *internalMockChannelRepository, service ChannelService) {
				mockInternalRepository.On("getChannel", ctx, mock.Anything, int64(84374)).Return(&YoutubeChannel{
					YoutubeChannelId: int64(84374),
					ExternalID:       "3263yw",
					Name:             "Some Channel",
				}, true, nil)
				mockInternalRepository.On("getChannelStat", ctx, mock.Anything, int64(84374)).Return([]YoutubeChannelStats{
					{
						YoutubeChannelID:     1,
						YoutubeChannelStatID: 84374,
						SubscribersCount:     5,
						CreatedAt:            helpers.ParseTime("2025-11-12T05:06:07Z"),
					},
				}, nil)

				channelStats, ok, err := service.GetChannelStats(ctx, int64(84374), nil, nil)
				require.NoError(t, err)
				require.NotNil(t, channelStats)
				assert.True(t, ok)
				assert.Equal(t, int64(84374), channelStats.ID)
				assert.Equal(t, "3263yw", channelStats.ChannelID)
				assert.Equal(t, 1, len(channelStats.Statistics))

				actualStats := channelStats.Statistics[0]
				assert.Equal(t, int64(5), actualStats.SubscribersCount)
				assert.Equal(t, helpers.ParseTime("2025-11-12T00:00:00Z"), actualStats.CreatedAt)

				mockInternalRepository.AssertExpectations(t)
			},
		},
	}

	mockTxtController := new(MockTxController)
	mockTxtController.On("Begin", ctx).Return(&MockTx{}, nil)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockInternalRepository := new(internalMockChannelRepository)
			service := NewYoutubeChannelService(mockTxtController, mockInternalRepository)

			tc.test(t, mockInternalRepository, service)
		})
	}
}
