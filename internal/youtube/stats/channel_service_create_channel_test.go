//go:build database

//nolint:testpackage //because of using direct queuer replacement for db tests isolation
package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestYoutubeChannelService_CreateChannel(t *testing.T) {
	ctx := t.Context()

	type testCase struct {
		name string
		test func(t *testing.T, service *YoutubeChannelService, repository *internalMockChannelRepository)
	}

	tests := []testCase{
		{
			"empty db", func(t *testing.T, service *YoutubeChannelService, repository *internalMockChannelRepository) {
				youtubeChannel := YoutubeChannel{
					ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
					Name:       "Google Developers",
				}
				repository.On("getChannelByExternalId", ctx, mock.Anything, "UC-lHJZR3Gqxm24_Vd_AJ5Yw").
					Return(YoutubeChannel{}, false, nil)
				repository.On("createChannel", ctx, mock.Anything, youtubeChannel).Return(
					YoutubeChannel{
						YoutubeChannelId: int64(345),
						ExternalID:       "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
						Name:             "Google Developers",
						CreatedAt:        time.Now(),
					}, nil)

				actualChannel, isNew, err := service.CreateChannel(ctx, youtubeChannel)
				require.NoError(t, err)
				require.NotNil(t, actualChannel.YoutubeChannelId)
				assert.True(t, isNew)

				assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", actualChannel.ExternalID)
				assert.Equal(t, "Google Developers", actualChannel.Name)
				assert.False(t, actualChannel.CreatedAt.IsZero())

				repository.AssertExpectations(t)
			},
		},
		{
			"channel exists", func(t *testing.T, service *YoutubeChannelService, repository *internalMockChannelRepository) {
				repository.On("getChannelByExternalId", ctx, mock.Anything, "UC-lHJZR3Gqxm24_Vd_AJ5Yw").
					Return(YoutubeChannel{
						YoutubeChannelId: 1,
						ExternalID:       "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
						Name:             "Google Developers",
					}, true, nil)

				channel := YoutubeChannel{
					ExternalID: "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
					Name:       "Google Developers",
				}

				existingChannel, created, err := service.CreateChannel(ctx, channel)
				assert.Nil(t, err)
				assert.False(t, created)

				assert.Equal(t, "UC-lHJZR3Gqxm24_Vd_AJ5Yw", existingChannel.ExternalID)
				assert.Equal(t, "Google Developers", existingChannel.Name)
				assert.Equal(t, int64(1), existingChannel.YoutubeChannelId)

				repository.AssertExpectations(t)
			},
		},
	}

	for _, tc := range tests {

		mockTxtController := new(MockTxController)
		mockTxtController.On("Begin", ctx).Return(&MockTx{}, nil)

		mockInternalRepository := new(internalMockChannelRepository)

		service := NewYoutubeChannelService(mockTxtController, mockInternalRepository)

		t.Run(tc.name, func(t *testing.T) {
			tc.test(t, service, mockInternalRepository)
		})
	}
}
