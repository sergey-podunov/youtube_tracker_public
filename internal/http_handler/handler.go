package http_handler

import (
	"context"
	"fmt"
	"time"
	"youtube_tracker/internal/api"
	"youtube_tracker/internal/youtube/stats"
)

type MainHttpHandler struct {
	api.UnimplementedHandler
	channelRepository *stats.ChannelRepository
}

func NewHttpHandler(channelRepository *stats.ChannelRepository) *MainHttpHandler {
	return &MainHttpHandler{
		channelRepository: channelRepository,
	}
}

func (s MainHttpHandler) StatisticsGeneratePost(ctx context.Context) (*api.StatGenerationStarted, error) {
	return &api.StatGenerationStarted{
		StatusPath: "/status",
	}, nil
}

var startTime = time.Now()

func (s MainHttpHandler) StatusGet(ctx context.Context) (*api.Status, error) {
	uptime := time.Since(startTime)
	return &api.Status{
		Uptime: api.NewOptString(fmt.Sprintf("%d ms", uptime.Milliseconds())),
	}, nil
}
