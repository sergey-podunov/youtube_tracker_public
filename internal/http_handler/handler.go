package http_handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"youtube_tracker/internal/api"
	"youtube_tracker/internal/youtube/stats"
)

type MainHttpHandler struct {
	api.UnimplementedHandler
	channelRepository *stats.YoutubeChannelRepository
}

func NewHttpHandler(channelRepository *stats.YoutubeChannelRepository) *MainHttpHandler {
	return &MainHttpHandler{
		channelRepository: channelRepository,
	}
}

func (handler MainHttpHandler) StatisticsGeneratePost(ctx context.Context) (*api.StatGenerationStarted, error) {
	return &api.StatGenerationStarted{
		StatusPath: "/status",
	}, nil
}

func (handler MainHttpHandler) YoutubeChannelPost(ctx context.Context, req *api.YoutubeChannel) (*api.YoutubeChannel, error) {
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

func (handler MainHttpHandler) YoutubeChannelIDGet(ctx context.Context, params api.YoutubeChannelIDGetParams) (api.YoutubeChannelIDGetRes, error) {
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

func (handler MainHttpHandler) StatusGet(ctx context.Context) (*api.Status, error) {
	uptime := time.Since(startTime)
	return &api.Status{
		Uptime: api.NewOptString(fmt.Sprintf("%d ms", uptime.Milliseconds())),
	}, nil
}
