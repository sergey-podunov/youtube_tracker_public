package http_handler

import (
	"context"
	"fmt"
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
}

func NewHTTPHandler(collector youtube.StatisticsCollector, channelService stats.ChannelService, channelRepository stats.ChannelRepository) *MainHTTPHandler {
	return &MainHTTPHandler{
		collector:         collector,
		channelService:    channelService,
		channelRepository: channelRepository,
	}
}

func (handler *MainHTTPHandler) StatisticsGeneratePost(ctx context.Context) (*api.StatGenerationStarted, error) {
	return &api.StatGenerationStarted{
		StatusPath: "/status",
	}, nil
}

func (handler *MainHTTPHandler) YoutubeChannelPost(ctx context.Context, req *api.YoutubeChannel) (api.YoutubeChannelPostRes, error) {
	channel := stats.YoutubeChannel{
		ExternalID: req.YoutubeID,
		Name:       req.Name,
	}

	createdChannel, isNew, err := handler.channelService.CreateChannel(ctx, channel)
	if err != nil {
		return nil, err
	}

	if !isNew {
		return &api.YoutubeChannelPostConflict{
			ID:        api.NewOptInt64(createdChannel.YoutubeChannelId),
			Name:      createdChannel.Name,
			YoutubeID: createdChannel.ExternalID,
			CreatedAt: api.NewOptString(createdChannel.CreatedAt.Format(time.RFC3339)),
		}, err
	}

	return &api.YoutubeChannelPostCreated{
		ID:        api.NewOptInt64(createdChannel.YoutubeChannelId),
		Name:      createdChannel.Name,
		YoutubeID: createdChannel.ExternalID,
		CreatedAt: api.NewOptString(createdChannel.CreatedAt.Format(time.RFC3339)),
	}, nil
}

func (handler *MainHTTPHandler) YoutubeChannelIDGet(ctx context.Context, params api.YoutubeChannelIDGetParams) (api.YoutubeChannelIDGetRes, error) {
	channel, ok, err := handler.channelRepository.GetChannel(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if !ok {
		return &api.YoutubeChannelIDGetNotFound{}, nil
	}

	return &api.YoutubeChannel{
		ID:        api.NewOptInt64(channel.YoutubeChannelId),
		Name:      channel.Name,
		YoutubeID: channel.ExternalID,
		CreatedAt: api.NewOptString(channel.CreatedAt.Format(time.RFC3339)),
	}, nil
}

var startTime = time.Now()

func (handler *MainHTTPHandler) StatusGet(ctx context.Context) (*api.Status, error) {
	uptime := time.Since(startTime)
	dockerTag := helpers.GetEnvWithFallback("BUILD_TAG", "undefined")

	return &api.Status{
		Uptime: api.NewOptString(fmt.Sprintf("%d ms", uptime.Milliseconds())),
		DockerTag: api.NewOptString(dockerTag),
	}, nil
}

func (handler *MainHTTPHandler) SchedulePost(ctx context.Context) (*api.StatGenerationStarted, error) {
	jobID := handler.collector.CollectStatistics(ctx)

	return &api.StatGenerationStarted{
		StatusPath: fmt.Sprintf("/schedule/job/%d", jobID),
	}, nil
}

func (handler *MainHTTPHandler) ScheduleJobIDGet(ctx context.Context, params api.ScheduleJobIDGetParams) (api.ScheduleJobIDGetRes, error) {
	job, err := handler.collector.GetJobStatus(params.ID)
	if err != nil {
		return nil, err
	}

	return &api.JobStatus{
		ID:     job.ID,
		Status: api.JobStatusStatus(job.Status),
		Total:  api.NewOptInt32(job.Total),
		Ready:  api.NewOptInt32(job.Ready),
		Error:  api.NewOptInt32(job.Error),
	}, nil
}

func (handler *MainHTTPHandler) YoutubeChannelIDStatisticsGet(
	ctx context.Context, params api.YoutubeChannelIDStatisticsGetParams) (api.YoutubeChannelIDStatisticsGetRes, error) {
	youtubeChannelID := params.ID

	channelStatsInfo, channelFound, err := handler.channelService.GetChannelStats(ctx, youtubeChannelID, nil, nil)
	if err != nil {
		return nil, err
	}

	if !channelFound {
		return &api.YoutubeChannelIDStatisticsGetNotFound{}, nil
	}

	statisticsItems := make([]api.YoutubeChannelStatisticsStatisticsItem, len(channelStatsInfo.Statistics))
	for i, statInfo := range channelStatsInfo.Statistics {
		statisticsItems[i] = api.YoutubeChannelStatisticsStatisticsItem{
			Date:        statInfo.CreatedAt,
			Subscribers: statInfo.SubscribersCount,
		}
	}

	return &api.YoutubeChannelStatistics{
		ID:         channelStatsInfo.ID,
		YoutubeID:  api.NewOptString(channelStatsInfo.ChannelID),
		Statistics: statisticsItems,
	}, nil
}
