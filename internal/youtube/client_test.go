//go:build thirdparty

package youtube

import (
	"log"
	"testing"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/notify"
	
	"github.com/stretchr/testify/require"
)

const youtube_channel_id = "UC16niRr50-MSBwiO3YDb3RA"
const youtube_channel_name = "@BBCNews"

func TestYoutubeClient(t *testing.T) {
	ctx := t.Context()

	rootPath, err := helpers.FindProjectRoot()
	if err != nil {
		t.Fatalf("failed to get absolute path to the root: %v", err)
	}

	client, err := NewHttpClient(ctx, notify.Noop{}, rootPath)
	require.NoError(t, err)

	t.Run("Test GetChannelId", func(t *testing.T) {
		channelId, err := client.GetChannelId(ctx, youtube_channel_name)
		require.NoError(t, err)
		require.Equal(t, youtube_channel_id, channelId)
		log.Printf("Channel ID: %s", channelId)
	})

	t.Run("Test GetChannelData", func(t *testing.T) {
		channelData, err := client.GetChannelData(ctx, youtube_channel_id)
		require.NoError(t, err)
		require.NotNil(t, channelData)
		require.True(t, channelData.SubscribersCount > 10000000)
		log.Printf("Subscribers count: %d", channelData.SubscribersCount)
	})

	log.Println("Test suite completed.")
}
