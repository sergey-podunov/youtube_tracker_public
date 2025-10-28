package http_handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"youtube_tracker/internal/api"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"
)

type MainHTTPHandler struct {
	api.UnimplementedHandler
	collector         *youtube.WorkerCollector
	channelRepository *stats.YoutubeChannelRepository
}

func NewHTTPHandler(collector *youtube.WorkerCollector, channelRepository *stats.YoutubeChannelRepository) *MainHTTPHandler {
	return &MainHTTPHandler{
		collector:         collector,
		channelRepository: channelRepository,
	}
}

func (handler *MainHTTPHandler) StatisticsGeneratePost(ctx context.Context) (*api.StatGenerationStarted, error) {
	return &api.StatGenerationStarted{
		StatusPath: "/status",
	}, nil
}

func (handler *MainHTTPHandler) YoutubeChannelPost(ctx context.Context, req *api.YoutubeChannel) (*api.YoutubeChannel, error) {
	channel := stats.YoutubeChannel{
		ExternalId: req.YoutubeID,
		Name:       req.Name,
	}

	createdChannel, err := handler.channelRepository.CreateChannel(ctx, channel)
	if err != nil {
		return nil, err
	}

	return &api.YoutubeChannel{
		ID:        api.NewOptInt64(createdChannel.YoutubeChannelId),
		Name:      createdChannel.Name,
		YoutubeID: createdChannel.ExternalId,
		CreatedAt: api.NewOptString(createdChannel.CreatedAt.Format(time.RFC3339)),
	}, nil
}

func (handler *MainHTTPHandler) YoutubeChannelIDGet(ctx context.Context, params api.YoutubeChannelIDGetParams) (api.YoutubeChannelIDGetRes, error) {
	channel, err := handler.channelRepository.GetChannel(ctx, params.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &api.YoutubeChannelIDGetNotFound{}, nil
		}

		return nil, err
	}

	return &api.YoutubeChannel{
		ID:        api.NewOptInt64(channel.YoutubeChannelId),
		Name:      channel.Name,
		YoutubeID: channel.ExternalId,
		CreatedAt: api.NewOptString(channel.CreatedAt.Format(time.RFC3339)),
	}, nil
}

var startTime = time.Now()

func (handler *MainHTTPHandler) StatusGet(ctx context.Context) (*api.Status, error) {
	uptime := time.Since(startTime)

	return &api.Status{
		Uptime: api.NewOptString(fmt.Sprintf("%d ms", uptime.Milliseconds())),
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
