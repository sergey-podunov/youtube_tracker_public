package http_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"youtube_tracker/internal/api"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"
	"youtube_tracker/internal/ytclient"
)

type MainHTTPHandler struct {
	api.UnimplementedHandler
	collector                   youtube.Collector
	channelJobController        youtube.JobController
	videoJobController          youtube.JobController
	videoDiscoveryJobController youtube.JobController
	channelService              stats.ChannelService
	videoService                stats.VideoService
	channelRepository           stats.ChannelRepository
	videoRepository             stats.VideoRepository
	logger                      *slog.Logger
}

func NewHTTPHandler(logger *slog.Logger,
	collector youtube.Collector,
	channelJobController youtube.JobController,
	videoJobController youtube.JobController,
	videoDiscoveryJobController youtube.JobController,
	channelService stats.ChannelService,
	videoService stats.VideoService,
	channelRepository stats.ChannelRepository,
	videoRepository stats.VideoRepository) *MainHTTPHandler {
	return &MainHTTPHandler{
		collector:                   collector,
		channelJobController:        channelJobController,
		videoJobController:          videoJobController,
		videoDiscoveryJobController: videoDiscoveryJobController,
		channelService:              channelService,
		videoService:                videoService,
		channelRepository:           channelRepository,
		videoRepository:             videoRepository,
		logger:                      logger.With(slog.String("component", "HTTPHandler")),
	}
}

func (handler *MainHTTPHandler) withRequestIdLoggerContext(ctx context.Context, fn func(ctx context.Context, logger *slog.Logger) error) error {
	loggerWithRequestID := helpers.LoggerWithRequestID(ctx, handler.logger)
	ctx = context.WithValue(ctx, helpers.LoggerKey, loggerWithRequestID)

	return fn(ctx, loggerWithRequestID)
}

func (handler *MainHTTPHandler) YoutubeChannelPost(ctx context.Context, req *api.YoutubeChannelRequest) (api.YoutubeChannelPostRes, error) {
	var resp api.YoutubeChannelPostRes

	err := handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		createdChannel, isNew, err := handler.channelService.CreateChannelFromURL(ctx, req.URL)
		if err != nil {
			var urlErr ytclient.ErrInvalidChannelURL
			if errors.As(err, &urlErr) {
				logger.Warn("Invalid channel URL", "url", req.URL, "err", err)
				resp = &api.YoutubeChannelClientError{Message: err.Error()}
				return nil
			}
			logger.Error("Failed to register channel", "url", req.URL, "err", err)
			return err
		}

		if !isNew {
			resp = &api.YoutubeChannelPostConflict{
				ID:        api.NewOptInt64(createdChannel.YoutubeChannelId),
				Title:     createdChannel.Title,
				YoutubeID: createdChannel.ExternalID,
				CreatedAt: api.NewOptString(createdChannel.CreatedAt.Format(time.RFC3339)),
			}
			return nil
		}

		resp = &api.YoutubeChannelPostCreated{
			ID:        api.NewOptInt64(createdChannel.YoutubeChannelId),
			Title:     createdChannel.Title,
			YoutubeID: createdChannel.ExternalID,
			CreatedAt: api.NewOptString(createdChannel.CreatedAt.Format(time.RFC3339)),
		}
		return nil
	})

	return resp, err
}

func (handler *MainHTTPHandler) YoutubeChannelIDGet(ctx context.Context, params api.YoutubeChannelIDGetParams) (api.YoutubeChannelIDGetRes, error) {
	var res api.YoutubeChannelIDGetRes

	err := handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		channel, ok, err := handler.channelRepository.GetChannel(ctx, params.ID)

		if err != nil {
			logger.Error("Error while getting chanel", "err", err, slog.Int64("channel_id", params.ID))
			return err
		}

		if !ok {
			logger.Info("Channel not found", slog.Int64("channel_id", params.ID))
			res = &api.YoutubeChannelIDGetNotFound{}
			return nil
		}

		res = &api.YoutubeChannel{
			ID:          api.NewOptInt64(channel.YoutubeChannelId),
			Title:       channel.Title,
			YoutubeID:   channel.ExternalID,
			Description: helpers.OptString(channel.Description),
			CustomURL:   helpers.OptString(channel.CustomURL),
			CreatedAt:   api.NewOptString(channel.CreatedAt.Format(time.RFC3339)),
		}
		return nil
	})

	return res, err
}

func (handler *MainHTTPHandler) YoutubeChannelsGet(ctx context.Context, params api.YoutubeChannelsGetParams) (*api.YoutubeChannelList, error) {
	page := params.Page.Value
	count := params.PageSize.Value
	channelsInfo, err := handler.channelService.GetChannels(ctx, page, count)
	if err != nil {
		return nil, err
	}

	channels := make([]api.YoutubeChannel, len(channelsInfo.Channels))
	for i, channel := range channelsInfo.Channels {
		channels[i] = api.YoutubeChannel{
			ID:          api.NewOptInt64(channel.YoutubeChannelId),
			Title:       channel.Title,
			Description: helpers.OptString(channel.Description),
			CustomURL:   helpers.OptString(channel.CustomURL),
			YoutubeID:   channel.ExternalID,
			CreatedAt:   api.NewOptString(channel.CreatedAt.Format(time.RFC3339)),
		}
	}

	return &api.YoutubeChannelList{
		CurrentPage: channelsInfo.CurrentPage,
		PageSize:    channelsInfo.PageSize,
		TotalPages:  channelsInfo.TotalPages,
		Channels:    channels,
	}, nil
}

var startTime = time.Now()

func (handler *MainHTTPHandler) StatusGet(ctx context.Context) (*api.Status, error) {
	uptime := time.Since(startTime)
	dockerTag := helpers.GetBuildTag()

	return &api.Status{
		Uptime:    api.NewOptString(fmt.Sprintf("%d ms", uptime.Milliseconds())),
		DockerTag: api.NewOptString(dockerTag),
	}, nil
}

func (handler *MainHTTPHandler) SchedulePost(ctx context.Context) (*api.StatGenerationStarted, error) {
	var jobID int64

	_ = handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		jobID = handler.collector.CollectStatistics(ctx, handler.channelJobController)
		return nil
	})

	return &api.StatGenerationStarted{
		StatusPath: fmt.Sprintf("/schedule/job/%d", jobID),
	}, nil
}

func (handler *MainHTTPHandler) ScheduleJobIDGet(ctx context.Context, params api.ScheduleJobIDGetParams) (api.ScheduleJobIDGetRes, error) {
	var resp api.ScheduleJobIDGetRes

	err := handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		job, err := handler.collector.GetJobStatus(params.ID)

		if err != nil {
			logger.Error("Error while getting job status", "err", err, slog.Int64("job_id", params.ID))
			return err
		}

		resp = &api.JobStatus{
			ID:     job.ID,
			Status: api.JobStatusStatus(job.Status),
			Total:  api.NewOptInt32(job.Total),
			Ready:  api.NewOptInt32(job.Ready),
			Error:  api.NewOptInt32(job.Error),
		}
		return nil
	})

	return resp, err
}

func (handler *MainHTTPHandler) YoutubeChannelIDStatisticsGet(
	ctx context.Context, params api.YoutubeChannelIDStatisticsGetParams) (api.YoutubeChannelIDStatisticsGetRes, error) {
	var resp api.YoutubeChannelIDStatisticsGetRes

	err := handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		youtubeChannelID := params.ID

		channelStatsInfo, channelFound, err := handler.channelService.GetChannelStats(ctx, youtubeChannelID, nil, nil)

		if err != nil {
			logger.Error("Error while getting channel stats", "err", err, slog.Int64("channel_id", youtubeChannelID))
			return err
		}

		if !channelFound {
			logger.Info("Channel not found", slog.Int64("channel_id", youtubeChannelID))
			resp = &api.YoutubeChannelIDStatisticsGetNotFound{}
			return nil
		}

		statisticsItems := make([]api.YoutubeChannelStatisticsStatisticsItem, len(channelStatsInfo.Statistics))
		for i, statInfo := range channelStatsInfo.Statistics {
			statisticsItems[i] = api.YoutubeChannelStatisticsStatisticsItem{
				Date:        statInfo.CreatedAt,
				Subscribers: statInfo.SubscribersCount,
			}
		}

		resp = &api.YoutubeChannelStatistics{
			ID:         channelStatsInfo.ID,
			YoutubeID:  api.NewOptString(channelStatsInfo.ChannelID),
			Statistics: statisticsItems,
		}
		return nil
	})

	return resp, err
}

func (handler *MainHTTPHandler) YoutubeChannelIDVideosGet(
	ctx context.Context, params api.YoutubeChannelIDVideosGetParams) (api.YoutubeChannelIDVideosGetRes, error) {
	var resp api.YoutubeChannelIDVideosGetRes

	err := handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		page := params.Page.Value
		count := params.PageSize.Value

		videosInfo, channelFound, err := handler.videoService.GetVideosByChannel(ctx, params.ID, page, count)
		if err != nil {
			logger.Error("Error while getting channel videos", "err", err, slog.Int64("channel_id", params.ID))
			return err
		}

		if !channelFound {
			logger.Info("Channel not found", slog.Int64("channel_id", params.ID))
			resp = &api.YoutubeChannelIDVideosGetNotFound{}
			return nil
		}

		videos := make([]api.YoutubeVideo, len(videosInfo.Videos))
		for i, video := range videosInfo.Videos {
			v := api.YoutubeVideo{
				ID:        api.NewOptInt64(video.YoutubeVideoId),
				ChannelID: api.NewOptInt64(video.YoutubeChannelId),
				Title:     video.Title,
				YoutubeID: video.ExternalID,
				CreatedAt: api.NewOptString(video.CreatedAt.Format(time.RFC3339)),
			}
			if video.PublishedAt != nil {
				v.PublishedAt = api.NewOptString(video.PublishedAt.Format(time.RFC3339))
			}
			videos[i] = v
		}

		resp = &api.YoutubeVideoList{
			CurrentPage: videosInfo.CurrentPage,
			PageSize:    videosInfo.PageSize,
			TotalPages:  videosInfo.TotalPages,
			Videos:      videos,
		}
		return nil
	})

	return resp, err
}

func (handler *MainHTTPHandler) YoutubeVideoIDGet(ctx context.Context, params api.YoutubeVideoIDGetParams) (api.YoutubeVideoIDGetRes, error) {
	var res api.YoutubeVideoIDGetRes

	err := handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		video, ok, err := handler.videoRepository.GetVideo(ctx, params.ID)

		if err != nil {
			logger.Error("Error while getting video", "err", err, slog.Int64("video_id", params.ID))
			return err
		}

		if !ok {
			logger.Info("Video not found", slog.Int64("video_id", params.ID))
			res = &api.YoutubeVideoIDGetNotFound{}
			return nil
		}

		v := &api.YoutubeVideo{
			ID:        api.NewOptInt64(video.YoutubeVideoId),
			ChannelID: api.NewOptInt64(video.YoutubeChannelId),
			Title:     video.Title,
			YoutubeID: video.ExternalID,
			CreatedAt: api.NewOptString(video.CreatedAt.Format(time.RFC3339)),
		}
		if video.PublishedAt != nil {
			v.PublishedAt = api.NewOptString(video.PublishedAt.Format(time.RFC3339))
		}
		res = v
		return nil
	})

	return res, err
}

func (handler *MainHTTPHandler) YoutubeVideoIDStatisticsGet(
	ctx context.Context, params api.YoutubeVideoIDStatisticsGetParams) (api.YoutubeVideoIDStatisticsGetRes, error) {
	var resp api.YoutubeVideoIDStatisticsGetRes

	err := handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		videoStatsInfo, videoFound, err := handler.videoService.GetVideoStats(ctx, params.ID)

		if err != nil {
			logger.Error("Error while getting video stats", "err", err, slog.Int64("video_id", params.ID))
			return err
		}

		if !videoFound {
			logger.Info("Video not found", slog.Int64("video_id", params.ID))
			resp = &api.YoutubeVideoIDStatisticsGetNotFound{}
			return nil
		}

		statisticsItems := make([]api.YoutubeVideoStatisticsStatisticsItem, len(videoStatsInfo.Statistics))
		for i, statInfo := range videoStatsInfo.Statistics {
			statisticsItems[i] = api.YoutubeVideoStatisticsStatisticsItem{
				Date:         statInfo.CreatedAt,
				ViewCount:    statInfo.ViewCount,
				LikeCount:    statInfo.LikeCount,
				CommentCount: statInfo.CommentCount,
			}
		}

		resp = &api.YoutubeVideoStatistics{
			ID:         videoStatsInfo.ID,
			YoutubeID:  api.NewOptString(videoStatsInfo.VideoID),
			Statistics: statisticsItems,
		}
		return nil
	})

	return resp, err
}

func (handler *MainHTTPHandler) ScheduleVideoDiscoveryPost(ctx context.Context) (*api.StatGenerationStarted, error) {
	var jobID int64

	_ = handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		jobID = handler.collector.CollectStatistics(ctx, handler.videoDiscoveryJobController)
		return nil
	})

	return &api.StatGenerationStarted{
		StatusPath: fmt.Sprintf("/schedule/video/discovery/job/%d", jobID),
	}, nil
}

func (handler *MainHTTPHandler) ScheduleVideoDiscoveryJobIDGet(ctx context.Context, params api.ScheduleVideoDiscoveryJobIDGetParams) (api.ScheduleVideoDiscoveryJobIDGetRes, error) {
	var resp api.ScheduleVideoDiscoveryJobIDGetRes

	err := handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		job, err := handler.collector.GetJobStatus(params.ID)

		if err != nil {
			logger.Error("Error while getting discovery job status", "err", err, slog.Int64("job_id", params.ID))
			return err
		}

		resp = &api.JobStatus{
			ID:     job.ID,
			Status: api.JobStatusStatus(job.Status),
			Total:  api.NewOptInt32(job.Total),
			Ready:  api.NewOptInt32(job.Ready),
			Error:  api.NewOptInt32(job.Error),
		}
		return nil
	})

	return resp, err
}

func (handler *MainHTTPHandler) ScheduleVideoPost(ctx context.Context) (*api.StatGenerationStarted, error) {
	var jobID int64

	_ = handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		jobID = handler.collector.CollectStatistics(ctx, handler.videoJobController)
		return nil
	})

	return &api.StatGenerationStarted{
		StatusPath: fmt.Sprintf("/schedule/video/job/%d", jobID),
	}, nil
}

func (handler *MainHTTPHandler) ScheduleVideoJobIDGet(ctx context.Context, params api.ScheduleVideoJobIDGetParams) (api.ScheduleVideoJobIDGetRes, error) {
	var resp api.ScheduleVideoJobIDGetRes

	err := handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		job, err := handler.collector.GetJobStatus(params.ID)

		if err != nil {
			logger.Error("Error while getting video job status", "err", err, slog.Int64("job_id", params.ID))
			return err
		}

		resp = &api.JobStatus{
			ID:     job.ID,
			Status: api.JobStatusStatus(job.Status),
			Total:  api.NewOptInt32(job.Total),
			Ready:  api.NewOptInt32(job.Ready),
			Error:  api.NewOptInt32(job.Error),
		}
		return nil
	})

	return resp, err
}
