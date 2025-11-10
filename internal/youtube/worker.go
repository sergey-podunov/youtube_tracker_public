package youtube

import (
	"context"
	"youtube_tracker/internal/youtube/stats"
)

type Worker interface {
	GetChannelStats(channelID int64) error
}

type StatisticsWorker struct {
	channelRep stats.ChannelRepository
	client     Client
}

func NewStatisticsWorker(channelRep stats.ChannelRepository, client Client) *StatisticsWorker {
	return &StatisticsWorker{
		channelRep: channelRep,
		client:     client,
	}
}

func (w *StatisticsWorker) GetChannelStats(channelID int64) error {
	ctx := context.Background()

	channel, _, err := w.channelRep.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}

	channelData, err := w.client.GetChannelData(ctx, channel.ExternalID)
	if err != nil {
		return err
	}

	_, err = w.channelRep.StoreSubscriptionsCount(ctx, stats.YoutubeChannelStats{
		YoutubeChannelID: channelID,
		SubscribersCount: channelData.SubscribersCount,
	})
	if err != nil {
		return err
	}

	return nil
}
