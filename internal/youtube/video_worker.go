package youtube

import (
	"context"
	"log/slog"
	"time"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/youtube/stats"
	"youtube_tracker/internal/ytclient"
)

const videoWorkerComponentName = "VideoWorker"

type VideoWorker struct {
	videoRepo  stats.VideoRepository
	client     ytclient.Client
	logger     *slog.Logger
	videoLimit int
}

func NewVideoWorker(logger *slog.Logger, videoRep stats.VideoRepository, client ytclient.Client, videoLimit int) *VideoWorker {
	return &VideoWorker{
		videoRepo:  videoRep,
		client:     client,
		logger:     logger.With(slog.String("component", videoWorkerComponentName)),
		videoLimit: videoLimit,
	}
}

func (w *VideoWorker) JobIDs(ctx context.Context) ([]int64, error) {
	videos, err := w.videoRepo.GetVideos(ctx, time.Now().UTC(), w.videoLimit)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(videos))
	for i, v := range videos {
		ids[i] = v.YoutubeVideoId
	}
	return ids, nil
}

func (w *VideoWorker) Execute(ctx context.Context, videoID int64) error {
	logger := helpers.LoggerFromContextWithDefault(ctx, videoWorkerComponentName, w.logger)

	logger.Info("Getting video stats", slog.Int64("video_id", videoID))
	video, _, err := w.videoRepo.GetVideo(ctx, videoID)
	if err != nil {
		return err
	}

	videosData, err := w.client.GetVideosData(ctx, []string{video.ExternalID})
	if err != nil {
		return err
	}

	if len(videosData) == 0 {
		logger.Warn("No data returned for video", slog.Int64("video_id", videoID), slog.String("external_id", video.ExternalID))
		return nil
	}

	videoData := videosData[0]

	_, err = w.videoRepo.StoreVideoStats(ctx, stats.YoutubeVideoStats{
		YoutubeVideoID: videoID,
		ViewCount:      videoData.ViewCount,
		LikeCount:      videoData.LikeCount,
		CommentCount:   videoData.CommentCount,
	})
	if err != nil {
		return err
	}

	return nil
}
