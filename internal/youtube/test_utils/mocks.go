package test_utils

import (
	"context"
	"youtube_tracker/internal/youtube"
	
	"github.com/stretchr/testify/mock"
)

type MockYoutubeClient struct {
	youtube.EmptyClient
	mock.Mock
}

func (m *MockYoutubeClient) GetChannelData(ctx context.Context, channelId string) (youtube.ChannelData, error) {
	args := m.Called(ctx, channelId)

	var data youtube.ChannelData
	if args.Get(0) != nil {
		data = args.Get(0).(youtube.ChannelData)
	}

	return data, args.Error(1)
}

type MockStatisticsWorker struct {
	mock.Mock
}

func (m *MockStatisticsWorker) GetChannelStats(ctx context.Context, channelID int64) error {
	args := m.Called(ctx, channelID)
	return args.Error(0)
}
