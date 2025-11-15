package youtube

import (
	"context"
	"log/slog"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/youtube/stats"
)

const statisticsWorkerComponentName = "StatisticsWorker"

type Worker interface {
	GetChannelStats(ctx context.Context, channelID int64) error
}

type StatisticsWorker struct {
	channelRep stats.ChannelRepository
	client     Client
	logger     *slog.Logger
}

func NewStatisticsWorker(logger *slog.Logger, channelRep stats.ChannelRepository, client Client) *StatisticsWorker {
	return &StatisticsWorker{
		channelRep: channelRep,
		client:     client,
		logger:     logger.With(slog.String("component", statisticsWorkerComponentName)),
	}
}

func (w *StatisticsWorker) GetChannelStats(ctx context.Context, channelID int64) error {
	logger := helpers.LoggerFromContext(ctx, statisticsCollectorComponentName, w.logger)
	
	logger.Info("Getting channel stats", slog.Int64("channel_id", channelID))
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
