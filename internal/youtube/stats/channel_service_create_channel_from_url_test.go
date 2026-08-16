package stats

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"youtube_tracker/internal/storage"
	"youtube_tracker/internal/ytclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestYoutubeChannelService_CreateChannelFromURL(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	const channelURL = "https://youtube.com/@BBCNews"

	t.Run("creates new channel from valid URL", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		tx := new(MockTxController)
		tx.On("Begin", ctx).Return(&MockTx{}, nil)

		data := ytclient.ChannelData{
			ChannelID:   "UC16niRr50-MSBwiO3YDb3RA",
			Title:       "BBC News",
			Description: "Breaking news",
			CustomURL:   "@BBCNews",
		}
		client.On("GetChannelByHandle", ctx, "@BBCNews").Return(data, nil)

		desc := data.Description
		cu := data.CustomURL
		expectedChannel := YoutubeChannel{
			ExternalID:  data.ChannelID,
			Title:       data.Title,
			Description: &desc,
			CustomURL:   &cu,
		}
		repo.On("getChannelByExternalId", ctx, mock.Anything, data.ChannelID).Return(YoutubeChannel{}, false, nil)
		repo.On("createChannel", ctx, mock.Anything, expectedChannel).Return(
			YoutubeChannel{
				YoutubeChannelId: 1,
				ExternalID:       data.ChannelID,
				Title:            data.Title,
				CreatedAt:        time.Now(),
			}, nil)

		service := NewYoutubeChannelService(logger, tx, repo, client, nil)
		channel, isNew, err := service.CreateChannelFromURL(ctx, channelURL)

		require.NoError(t, err)
		assert.True(t, isNew)
		assert.Equal(t, data.ChannelID, channel.ExternalID)
		assert.Equal(t, data.Title, channel.Title)
		client.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("returns existing channel when URL resolves to duplicate", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		tx := new(MockTxController)
		tx.On("Begin", ctx).Return(&MockTx{}, nil)

		data := ytclient.ChannelData{ChannelID: "UC16niRr50-MSBwiO3YDb3RA", Title: "BBC News"}
		client.On("GetChannelByHandle", ctx, "@BBCNews").Return(data, nil)

		existing := YoutubeChannel{YoutubeChannelId: 5, ExternalID: data.ChannelID, Title: data.Title}
		repo.On("getChannelByExternalId", ctx, mock.Anything, data.ChannelID).Return(existing, true, nil)

		service := NewYoutubeChannelService(logger, tx, repo, client, nil)
		channel, isNew, err := service.CreateChannelFromURL(ctx, channelURL)

		require.NoError(t, err)
		assert.False(t, isNew)
		assert.Equal(t, int64(5), channel.YoutubeChannelId)
		client.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("creates channel from channel ID URL", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		tx := new(MockTxController)
		tx.On("Begin", ctx).Return(&MockTx{}, nil)

		data := ytclient.ChannelData{
			ChannelID:   "UC-lHJZR3Gqxm24_Vd_AJ5Yw",
			Title:       "Google Developers",
			Description: "The Google Developers channel",
			CustomURL:   "@googledevelopers",
		}
		client.On("GetChannelData", ctx, "UC-lHJZR3Gqxm24_Vd_AJ5Yw").Return(data, nil)

		desc := data.Description
		cu := data.CustomURL
		expectedChannel := YoutubeChannel{
			ExternalID:  data.ChannelID,
			Title:       data.Title,
			Description: &desc,
			CustomURL:   &cu,
		}
		repo.On("getChannelByExternalId", ctx, mock.Anything, data.ChannelID).Return(YoutubeChannel{}, false, nil)
		repo.On("createChannel", ctx, mock.Anything, expectedChannel).Return(
			YoutubeChannel{
				YoutubeChannelId: 2,
				ExternalID:       data.ChannelID,
				Title:            data.Title,
				CreatedAt:        time.Now(),
			}, nil)

		service := NewYoutubeChannelService(logger, tx, repo, client, nil)
		channel, isNew, err := service.CreateChannelFromURL(ctx, "https://youtube.com/channel/UC-lHJZR3Gqxm24_Vd_AJ5Yw")

		require.NoError(t, err)
		assert.True(t, isNew)
		assert.Equal(t, data.ChannelID, channel.ExternalID)
		client.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("creates channel from legacy user URL", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		tx := new(MockTxController)
		tx.On("Begin", ctx).Return(&MockTx{}, nil)

		data := ytclient.ChannelData{
			ChannelID:   "UC16niRr50-MSBwiO3YDb3RA",
			Title:       "BBC News",
			Description: "BBC News channel",
			CustomURL:   "@BBCNews",
		}
		client.On("GetChannelByUsername", ctx, "BBCNews").Return(data, nil)

		desc := data.Description
		cu := data.CustomURL
		expectedChannel := YoutubeChannel{
			ExternalID:  data.ChannelID,
			Title:       data.Title,
			Description: &desc,
			CustomURL:   &cu,
		}
		repo.On("getChannelByExternalId", ctx, mock.Anything, data.ChannelID).Return(YoutubeChannel{}, false, nil)
		repo.On("createChannel", ctx, mock.Anything, expectedChannel).Return(
			YoutubeChannel{
				YoutubeChannelId: 3,
				ExternalID:       data.ChannelID,
				Title:            data.Title,
				CreatedAt:        time.Now(),
			}, nil)

		service := NewYoutubeChannelService(logger, tx, repo, client, nil)
		channel, isNew, err := service.CreateChannelFromURL(ctx, "https://youtube.com/user/BBCNews")

		require.NoError(t, err)
		assert.True(t, isNew)
		assert.Equal(t, data.ChannelID, channel.ExternalID)
		client.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("propagates client error", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		tx := new(MockTxController)

		client.On("GetChannelByHandle", ctx, "@BBCNews").Return(nil, errors.New("api error"))

		service := NewYoutubeChannelService(logger, tx, repo, client, nil)
		_, _, err := service.CreateChannelFromURL(ctx, channelURL)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "api error")
		client.AssertExpectations(t)
	})

	t.Run("returns ErrInvalidChannelURL for bad URL", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		tx := new(MockTxController)

		service := NewYoutubeChannelService(logger, tx, repo, client, nil)
		_, _, err := service.CreateChannelFromURL(ctx, "https://youtube.com/c/SomeName")

		require.Error(t, err)
		var urlErr ytclient.ErrInvalidChannelURL
		assert.ErrorAs(t, err, &urlErr)
		client.AssertNotCalled(t, mock.Anything)
	})

	t.Run("fetches and stores thumbnail when new channel created", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		fetcher := new(storage.MockFileFetcher)
		tx := new(MockTxController)
		tx.On("Begin", ctx).Return(&MockTx{}, nil)

		data := ytclient.ChannelData{
			ChannelID:    "UC16niRr50-MSBwiO3YDb3RA",
			Title:        "BBC News",
			Description:  "Breaking news",
			CustomURL:    "@BBCNews",
			ThumbnailURL: "https://yt3.googleusercontent.com/thumb.jpg",
		}
		client.On("GetChannelByHandle", ctx, "@BBCNews").Return(data, nil)

		desc := data.Description
		cu := data.CustomURL
		expectedChannel := YoutubeChannel{
			ExternalID:  data.ChannelID,
			Title:       data.Title,
			Description: &desc,
			CustomURL:   &cu,
		}
		repo.On("getChannelByExternalId", ctx, mock.Anything, data.ChannelID).Return(YoutubeChannel{}, false, nil)
		repo.On("createChannel", ctx, mock.Anything, expectedChannel).Return(
			YoutubeChannel{
				YoutubeChannelId: 1,
				ExternalID:       data.ChannelID,
				Title:            data.Title,
				CreatedAt:        time.Now(),
			}, nil)

		fetcher.On("FetchAndStore", ctx, "https://yt3.googleusercontent.com/thumb.jpg", "/youtube/channel/UC16niRr50-MSBwiO3YDb3RA.jpg").Return(nil)

		service := NewYoutubeChannelService(logger, tx, repo, client, fetcher)
		channel, isNew, err := service.CreateChannelFromURL(ctx, channelURL)

		require.NoError(t, err)
		assert.True(t, isNew)
		assert.Equal(t, data.ChannelID, channel.ExternalID)
		fetcher.AssertExpectations(t)
	})

	t.Run("thumbnail fetch error does not fail channel creation", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		fetcher := new(storage.MockFileFetcher)
		tx := new(MockTxController)
		tx.On("Begin", ctx).Return(&MockTx{}, nil)

		data := ytclient.ChannelData{
			ChannelID:    "UC16niRr50-MSBwiO3YDb3RA",
			Title:        "BBC News",
			Description:  "Breaking news",
			CustomURL:    "@BBCNews",
			ThumbnailURL: "https://yt3.googleusercontent.com/thumb.jpg",
		}
		client.On("GetChannelByHandle", ctx, "@BBCNews").Return(data, nil)

		desc := data.Description
		cu := data.CustomURL
		expectedChannel := YoutubeChannel{
			ExternalID:  data.ChannelID,
			Title:       data.Title,
			Description: &desc,
			CustomURL:   &cu,
		}
		repo.On("getChannelByExternalId", ctx, mock.Anything, data.ChannelID).Return(YoutubeChannel{}, false, nil)
		repo.On("createChannel", ctx, mock.Anything, expectedChannel).Return(
			YoutubeChannel{
				YoutubeChannelId: 1,
				ExternalID:       data.ChannelID,
				Title:            data.Title,
				CreatedAt:        time.Now(),
			}, nil)

		fetcher.On("FetchAndStore", ctx, "https://yt3.googleusercontent.com/thumb.jpg", "/youtube/channel/UC16niRr50-MSBwiO3YDb3RA.jpg").Return(fmt.Errorf("s3 upload failed"))

		service := NewYoutubeChannelService(logger, tx, repo, client, fetcher)
		channel, isNew, err := service.CreateChannelFromURL(ctx, channelURL)

		require.NoError(t, err)
		assert.True(t, isNew)
		assert.Equal(t, data.ChannelID, channel.ExternalID)
		fetcher.AssertExpectations(t)
	})

	t.Run("skips thumbnail fetch when URL is empty", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		fetcher := new(storage.MockFileFetcher)
		tx := new(MockTxController)
		tx.On("Begin", ctx).Return(&MockTx{}, nil)

		data := ytclient.ChannelData{
			ChannelID:   "UC16niRr50-MSBwiO3YDb3RA",
			Title:       "BBC News",
			Description: "Breaking news",
			CustomURL:   "@BBCNews",
		}
		client.On("GetChannelByHandle", ctx, "@BBCNews").Return(data, nil)

		desc := data.Description
		cu := data.CustomURL
		expectedChannel := YoutubeChannel{
			ExternalID:  data.ChannelID,
			Title:       data.Title,
			Description: &desc,
			CustomURL:   &cu,
		}
		repo.On("getChannelByExternalId", ctx, mock.Anything, data.ChannelID).Return(YoutubeChannel{}, false, nil)
		repo.On("createChannel", ctx, mock.Anything, expectedChannel).Return(
			YoutubeChannel{
				YoutubeChannelId: 1,
				ExternalID:       data.ChannelID,
				Title:            data.Title,
				CreatedAt:        time.Now(),
			}, nil)

		service := NewYoutubeChannelService(logger, tx, repo, client, fetcher)
		_, isNew, err := service.CreateChannelFromURL(ctx, channelURL)

		require.NoError(t, err)
		assert.True(t, isNew)
		fetcher.AssertNotCalled(t, "FetchAndStore", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("skips thumbnail fetch for existing channel", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		fetcher := new(storage.MockFileFetcher)
		tx := new(MockTxController)
		tx.On("Begin", ctx).Return(&MockTx{}, nil)

		data := ytclient.ChannelData{
			ChannelID:    "UC16niRr50-MSBwiO3YDb3RA",
			Title:        "BBC News",
			ThumbnailURL: "https://yt3.googleusercontent.com/thumb.jpg",
		}
		client.On("GetChannelByHandle", ctx, "@BBCNews").Return(data, nil)

		existing := YoutubeChannel{YoutubeChannelId: 5, ExternalID: data.ChannelID, Title: data.Title}
		repo.On("getChannelByExternalId", ctx, mock.Anything, data.ChannelID).Return(existing, true, nil)

		service := NewYoutubeChannelService(logger, tx, repo, client, fetcher)
		channel, isNew, err := service.CreateChannelFromURL(ctx, channelURL)

		require.NoError(t, err)
		assert.False(t, isNew)
		assert.Equal(t, int64(5), channel.YoutubeChannelId)
		fetcher.AssertNotCalled(t, "FetchAndStore", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("works without fileFetcher (nil)", func(t *testing.T) {
		client := new(MockYoutubeClient)
		repo := new(internalMockChannelRepository)
		tx := new(MockTxController)
		tx.On("Begin", ctx).Return(&MockTx{}, nil)

		data := ytclient.ChannelData{
			ChannelID:    "UC16niRr50-MSBwiO3YDb3RA",
			Title:        "BBC News",
			Description:  "Breaking news",
			CustomURL:    "@BBCNews",
			ThumbnailURL: "https://yt3.googleusercontent.com/thumb.jpg",
		}
		client.On("GetChannelByHandle", ctx, "@BBCNews").Return(data, nil)

		desc := data.Description
		cu := data.CustomURL
		expectedChannel := YoutubeChannel{
			ExternalID:  data.ChannelID,
			Title:       data.Title,
			Description: &desc,
			CustomURL:   &cu,
		}
		repo.On("getChannelByExternalId", ctx, mock.Anything, data.ChannelID).Return(YoutubeChannel{}, false, nil)
		repo.On("createChannel", ctx, mock.Anything, expectedChannel).Return(
			YoutubeChannel{
				YoutubeChannelId: 1,
				ExternalID:       data.ChannelID,
				Title:            data.Title,
				CreatedAt:        time.Now(),
			}, nil)

		service := NewYoutubeChannelService(logger, tx, repo, client, nil)
		channel, isNew, err := service.CreateChannelFromURL(ctx, channelURL)

		require.NoError(t, err)
		assert.True(t, isNew)
		assert.Equal(t, data.ChannelID, channel.ExternalID)
	})
}
