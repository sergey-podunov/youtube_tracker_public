package http_handler

import (
	"context"
	"youtube_tracker/internal/api"
)

type StatisticsHttpHandler struct {
	api.UnimplementedHandler
}

func (s StatisticsHttpHandler) StatisticsGeneratePost(ctx context.Context) (*api.StatGenerationStarted, error) {
	return &api.StatGenerationStarted{
		StatusPath: "/status",
	}, nil
}
