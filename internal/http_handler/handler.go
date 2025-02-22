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
	stats *stats.ChannelRepository
}

func (s MainHttpHandler) StatisticsGeneratePost(ctx context.Context) (*api.StatGenerationStarted, error) {
	return &api.StatGenerationStarted{
		StatusPath: "/status",
	}, nil
}

var startTime = time.Now()

func (s MainHttpHandler) StateGet(ctx context.Context) (*api.State, error) {
	uptime := time.Since(startTime)
	return &api.State{
		Uptime: api.NewOptString(fmt.Sprintf("%d ms", uptime.Milliseconds())),
	}, nil
}
