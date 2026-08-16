//go:build thirdparty

package youtube

import (
	"log"
	"os"
	"testing"
	"youtube_tracker/internal/notify"
	"youtube_tracker/internal/ytclient"

	"github.com/stretchr/testify/require"
)

const bbcChannelId = "UC16niRr50-MSBwiO3YDb3RA"
const bbcChannelHandle = "@BBCNews"

const pewChannelId = "UC-lHJZR3Gqxm24_Vd_AJ5Yw"
const pewChannelName = "PewDiePie"

func TestYoutubeClient(t *testing.T) {
	ctx := t.Context()

	googleApiKey := os.Getenv("GOOGLE_API_KEY")
	client, err := ytclient.NewHttpClient(ctx, notify.Noop{}, googleApiKey)
	require.NoError(t, err)

	t.Run("Test GetChannelId by handle", func(t *testing.T) {
		channelData, err := client.GetChannelByHandle(ctx, bbcChannelHandle)
		require.NoError(t, err)

		channelID := channelData.ChannelID
		require.Equal(t, bbcChannelId, channelID)
		log.Printf("Channel ID: %s", channelID)
	})

	t.Run("Test GetChannelId by username", func(t *testing.T) {
		channelData, err := client.GetChannelByUsername(ctx, pewChannelName)
		require.NoError(t, err)

		channelID := channelData.ChannelID
		require.Equal(t, pewChannelId, channelID)
		log.Printf("Channel ID: %s", channelID)
	})

	t.Run("Test GetChannelData", func(t *testing.T) {
		channelData, err := client.GetChannelData(ctx, bbcChannelId)
		require.NoError(t, err)
		require.NotNil(t, channelData)
		require.True(t, channelData.SubscribersCount > 10000000)
		require.True(t, len(channelData.Title) > 0)
		require.True(t, len(channelData.CustomURL) > 0)
		log.Printf("Subscribers count: %d", channelData.SubscribersCount)
	})

	t.Run("Test GetChannelVideos", func(t *testing.T) {
		videos, err := client.GetChannelVideos(ctx, bbcChannelId, 5)
		require.NoError(t, err)
		require.NotEmpty(t, videos)

		for _, video := range videos {
			require.NotEmpty(t, video.VideoID)
			require.NotEmpty(t, video.Title)
			require.Equal(t, bbcChannelId, video.ChannelID)
			require.NotEmpty(t, video.PublishedAt)
			require.True(t, video.ViewCount > 0)
		}
		log.Printf("Videos found: %d", len(videos))
	})

	log.Println("Test suite completed.")
}
