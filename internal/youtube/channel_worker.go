package youtube

import (
	"context"
	"log/slog"
	"time"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/youtube/stats"
	"youtube_tracker/internal/ytclient"
)

const channelWorkerComponentName = "ChannelWorker"

type ChannelWorker struct {
	channelRepo  stats.ChannelRepository
	client       ytclient.Client
	logger       *slog.Logger
	channelLimit int
}

func NewChannelWorker(logger *slog.Logger, channelRep stats.ChannelRepository, client ytclient.Client, channelLimit int) *ChannelWorker {
	return &ChannelWorker{
		channelRepo:  channelRep,
		client:       client,
		logger:       logger.With(slog.String("component", channelWorkerComponentName)),
		channelLimit: channelLimit,
	}
}

func (w *ChannelWorker) JobIDs(ctx context.Context) ([]int64, error) {
	channels, err := w.channelRepo.GetChannels(ctx, time.Now().UTC(), w.channelLimit)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(channels))
	for i, ch := range channels {
		ids[i] = ch.YoutubeChannelId
	}
	return ids, nil
}

func (w *ChannelWorker) Execute(ctx context.Context, channelID int64) error {
	logger := helpers.LoggerFromContextWithDefault(ctx, channelWorkerComponentName, w.logger)

	logger.Info("Getting channel stats", slog.Int64("channel_id", channelID))
	channel, _, err := w.channelRepo.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}

	channelData, err := w.client.GetChannelData(ctx, channel.ExternalID)
	if err != nil {
		return err
	}

	_, err = w.channelRepo.StoreSubscriptionsCount(ctx, stats.YoutubeChannelStats{
		YoutubeChannelID: channelID,
		SubscribersCount: channelData.SubscribersCount,
	})
	if err != nil {
		return err
	}

	return nil
}
