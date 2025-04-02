package youtube

import (
	"context"
	"youtube_tracker/internal/youtube/stats"
)

type YoutubeChannelWorker struct {
	channelRep stats.ChannelRepository
	client     Client
}

func (w *YoutubeChannelWorker) GetChannelStats(channelID int64) error {
	ctx := context.Background()
	channel, err := w.channelRep.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}

	channelData, err := w.client.GetChannelData(channel.ExternalId)
	if err != nil {
		return err
	}

	_, err = w.channelRep.StoreSubscriptionsCount(ctx, stats.YoutubeChannelStats{
		YoutubeChannelId: channelID,
		SubscribersCount: channelData.SubscribersCount,
	})
	if err != nil {
		return err
	}

	return nil
}
