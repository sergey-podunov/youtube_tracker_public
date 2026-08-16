package stats

import (
	"context"
	"log/slog"
	"math"
	"time"
	"youtube_tracker/internal/helpers"

	"github.com/jackc/pgx/v5"
)

type VideoService interface {
	CreateVideo(ctx context.Context, video YoutubeVideo) (YoutubeVideo, bool, error)
	GetVideoStats(ctx context.Context, videoID int64) (YoutubeVideoStatsInfo, bool, error)
	GetVideosByChannel(ctx context.Context, channelID int64, page int, pageSize int) (YoutubeVideosInfo, bool, error)
}

type YoutubeVideoStatsInfo struct {
	ID         int64
	VideoID    string
	Statistics []YoutubeVideoStats
}

type YoutubeVideosInfo struct {
	CurrentPage int
	TotalPages  int
	PageSize    int
	Videos      []YoutubeVideo
}

type YoutubeVideoService struct {
	db                helpers.TxController
	videoRepository   internalVideoRepository
	channelRepository internalChannelRepository
	logger            *slog.Logger
}

const videoServiceComponentName = "YoutubeVideoService"

func NewYoutubeVideoService(logger *slog.Logger, db helpers.TxController, videoRepository internalVideoRepository, channelRepository internalChannelRepository) *YoutubeVideoService {
	return &YoutubeVideoService{
		db:                db,
		videoRepository:   videoRepository,
		channelRepository: channelRepository,
		logger:            logger.With(slog.String("component", videoServiceComponentName)),
	}
}

func (s *YoutubeVideoService) CreateVideo(ctx context.Context, video YoutubeVideo) (YoutubeVideo, bool, error) {
	var createdVideo YoutubeVideo
	var videoCreated bool

	logger := helpers.LoggerFromContextWithDefault(ctx, videoServiceComponentName, s.logger)
	err := helpers.RunInTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		_, channelExists, err := s.channelRepository.getChannel(ctx, tx, video.YoutubeChannelId)
		if err != nil {
			return err
		}

		if !channelExists {
			logger.Warn("Channel not found for video", slog.Int64("channel_id", video.YoutubeChannelId))
			return nil
		}

		existingVideo, ok, err := s.videoRepository.getVideoByExternalId(ctx, tx, video.ExternalID)
		if err != nil {
			return err
		}

		if ok {
			logger.Debug("Video already exists", slog.Any("video", existingVideo))
			createdVideo = existingVideo
			return nil
		}

		newVideo, err := s.videoRepository.createVideo(ctx, tx, video)
		if err != nil {
			return err
		}

		createdVideo = newVideo
		videoCreated = true

		logger.Info("Video created", slog.Any("video", createdVideo))
		return nil
	})

	return createdVideo, videoCreated, err
}

func (s *YoutubeVideoService) GetVideoStats(ctx context.Context, videoID int64) (YoutubeVideoStatsInfo, bool, error) {
	logger := helpers.LoggerFromContextWithDefault(ctx, videoServiceComponentName, s.logger)

	var statsInfo YoutubeVideoStatsInfo
	var ok bool
	err := helpers.RunInTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		video, videoExists, err := s.videoRepository.getVideo(ctx, tx, videoID)
		if err != nil {
			return err
		}

		if !videoExists {
			logger.Info("Video not found", slog.Int64("video_id", videoID))
			ok = false
			return nil
		}

		videoStats, err := s.videoRepository.getVideoStat(ctx, tx, videoID)
		if err != nil {
			return err
		}

		if len(videoStats) == 0 {
			logger.Info("There is no stat for video", slog.Int64("video_id", videoID))
		}

		for i := range videoStats {
			videoStats[i].CreatedAt = videoStats[i].CreatedAt.Truncate(time.Hour * 24)
		}

		statsInfo = YoutubeVideoStatsInfo{
			ID:         video.YoutubeVideoId,
			VideoID:    video.ExternalID,
			Statistics: videoStats,
		}
		ok = true

		return nil
	})

	return statsInfo, ok, err
}

func (s *YoutubeVideoService) GetVideosByChannel(ctx context.Context, channelID int64, page int, pageSize int) (YoutubeVideosInfo, bool, error) {
	var videosInfo YoutubeVideosInfo
	var channelFound bool

	logger := helpers.LoggerFromContextWithDefault(ctx, videoServiceComponentName, s.logger)
	err := helpers.RunInTx(ctx, s.db, func(ctx context.Context, tx pgx.Tx) error {
		_, channelExists, err := s.channelRepository.getChannel(ctx, tx, channelID)
		if err != nil {
			return err
		}

		if !channelExists {
			logger.Info("Channel not found", slog.Int64("channel_id", channelID))
			channelFound = false
			return nil
		}

		channelFound = true

		offset := getOffsetByPage(page, pageSize)
		pageSize = getPageSize(pageSize)

		videos, err := s.videoRepository.getVideosByChannelPaginated(ctx, tx, channelID, offset, pageSize)
		if err != nil {
			return err
		}

		videosCount, err := s.videoRepository.getVideosByChannelCount(ctx, tx, channelID)
		if err != nil {
			return err
		}

		currentPage := getCurrentPage(page)
		totalPages := int(math.Ceil(float64(videosCount) / float64(pageSize)))

		videosInfo = YoutubeVideosInfo{
			CurrentPage: currentPage,
			TotalPages:  totalPages,
			PageSize:    pageSize,
			Videos:      videos,
		}
		return nil
	})

	return videosInfo, channelFound, err
}
