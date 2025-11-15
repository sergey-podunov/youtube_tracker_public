package http_handler

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"youtube_tracker/internal/api"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"
)

type MainHTTPHandler struct {
	api.UnimplementedHandler
	collector         youtube.StatisticsCollector
	channelService    stats.ChannelService
	channelRepository stats.ChannelRepository
	logger            *slog.Logger
}

func NewHTTPHandler(logger *slog.Logger, collector youtube.StatisticsCollector, channelService stats.ChannelService, channelRepository stats.ChannelRepository) *MainHTTPHandler {
	return &MainHTTPHandler{
		collector:         collector,
		channelService:    channelService,
		channelRepository: channelRepository,
		logger:            logger.With(slog.String("component", "HTTPHandler")),
	}
}

func (handler *MainHTTPHandler) withRequestIdLoggerContext(ctx context.Context, fn func(ctx context.Context, logger *slog.Logger) error) error {
	loggerWithRequestID := helpers.LoggerWithRequestID(ctx, handler.logger)
	ctx = context.WithValue(ctx, helpers.LoggerKey, loggerWithRequestID)

	return fn(ctx, loggerWithRequestID)
}

func (handler *MainHTTPHandler) StatisticsGeneratePost(ctx context.Context) (*api.StatGenerationStarted, error) {
	return &api.StatGenerationStarted{
		StatusPath: "/status",
	}, nil
}

func (handler *MainHTTPHandler) YoutubeChannelPost(ctx context.Context, req *api.YoutubeChannel) (api.YoutubeChannelPostRes, error) {
	var resp api.YoutubeChannelPostRes

	err := handler.withRequestIdLoggerContext(ctx, func(ctx context.Context, logger *slog.Logger) error {
		channel := stats.YoutubeChannel{
			ExternalID: req.YoutubeID,
			Name:       req.Name,
		}
		createdChannel, isNew, err := handler.channelService.CreateChannel(ctx, channel)

		if err != nil {
			logger.Error("Error while creating channel", "err", err, slog.Any("channel", channel))
			return err
		}

		if !isNew {
			resp = &api.YoutubeChannelPostConflict{
				ID:        api.NewOptInt64(createdChannel.YoutubeChannelId),
				Name:      createdChannel.Name,
				YoutubeID: createdChannel.ExternalID,
				CreatedAt: api.NewOptString(createdChannel.CreatedAt.Format(time.RFC3339)),
			}
			return nil
		}

		resp = &api.YoutubeChannelPostCreated{
			ID:        api.NewOptInt64(createdChannel.YoutubeChannelId),
			Name:      createdChannel.Name,
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
			ID:        api.NewOptInt64(channel.YoutubeChannelId),
			Name:      channel.Name,
			YoutubeID: channel.ExternalID,
			CreatedAt: api.NewOptString(channel.CreatedAt.Format(time.RFC3339)),
		}
		return nil
	})

	return res, err
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
		jobID = handler.collector.CollectStatistics(ctx)
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
