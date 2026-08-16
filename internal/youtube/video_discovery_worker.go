package youtube

import (
	"context"
	"log/slog"
	"time"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/youtube/stats"
	"youtube_tracker/internal/ytclient"
)

const videoDiscoveryWorkerComponentName = "VideoDiscoveryWorker"

type VideoDiscoveryWorker struct {
	channelRep      stats.ChannelRepository
	videoService    stats.VideoService
	client          ytclient.Client
	videoFetchLimit int64
	channelLimit    int
	logger          *slog.Logger
}

func NewVideoDiscoveryWorker(logger *slog.Logger, channelRep stats.ChannelRepository, videoService stats.VideoService, client ytclient.Client, videoFetchLimit int64, channelLimit int) *VideoDiscoveryWorker {
	return &VideoDiscoveryWorker{
		channelRep:      channelRep,
		videoService:    videoService,
		client:          client,
		videoFetchLimit: videoFetchLimit,
		channelLimit:    channelLimit,
		logger:          logger.With(slog.String("component", videoDiscoveryWorkerComponentName)),
	}
}

func (w *VideoDiscoveryWorker) JobIDs(ctx context.Context) ([]int64, error) {
	channels, err := w.channelRep.GetChannels(ctx, time.Now().UTC(), w.channelLimit)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(channels))
	for i, ch := range channels {
		ids[i] = ch.YoutubeChannelId
	}
	return ids, nil
}

func (w *VideoDiscoveryWorker) Execute(ctx context.Context, channelID int64) error {
	logger := helpers.LoggerFromContextWithDefault(ctx, videoDiscoveryWorkerComponentName, w.logger)

	logger.Info("Discovering channel videos", slog.Int64("channel_id", channelID))
	channel, ok, err := w.channelRep.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}

	if !ok {
		logger.Warn("Channel not found", slog.Int64("channel_id", channelID))
		return nil
	}

	videosData, err := w.client.GetChannelVideos(ctx, channel.ExternalID, w.videoFetchLimit)
	if err != nil {
		return err
	}

	created := 0
	for _, videoData := range videosData {
		video := stats.YoutubeVideo{
			YoutubeChannelId: channelID,
			ExternalID:       videoData.VideoID,
			Title:            videoData.Title,
		}

		publishedAt, parseErr := time.Parse(time.RFC3339, videoData.PublishedAt)
		if parseErr != nil {
			logger.Warn("Failed to parse video published_at",
				slog.String("external_id", videoData.VideoID), slog.String("published_at", videoData.PublishedAt), "err", parseErr)
		} else {
			video.PublishedAt = &publishedAt
		}

		_, isNew, err := w.videoService.CreateVideo(ctx, video)
		if err != nil {
			return err
		}

		if isNew {
			created++
		}
	}

	logger.Info("Channel videos discovered",
		slog.Int64("channel_id", channelID), slog.Int("total", len(videosData)), slog.Int("new", created))
	return nil
}
