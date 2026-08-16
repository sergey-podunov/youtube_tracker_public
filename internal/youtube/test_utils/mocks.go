package test_utils

import (
	"context"
	"youtube_tracker/internal/ytclient"

	"github.com/stretchr/testify/mock"
)

type MockYoutubeClient struct {
	mock.Mock
}

func (m *MockYoutubeClient) GetChannelData(ctx context.Context, channelId string) (ytclient.ChannelData, error) {
	args := m.Called(ctx, channelId)

	var data ytclient.ChannelData
	if args.Get(0) != nil {
		data = args.Get(0).(ytclient.ChannelData)
	}

	return data, args.Error(1)
}

func (m *MockYoutubeClient) GetChannelByHandle(ctx context.Context, handle string) (ytclient.ChannelData, error) {
	args := m.Called(ctx, handle)

	var data ytclient.ChannelData
	if args.Get(0) != nil {
		data = args.Get(0).(ytclient.ChannelData)
	}

	return data, args.Error(1)
}

func (m *MockYoutubeClient) GetChannelByUsername(ctx context.Context, username string) (ytclient.ChannelData, error) {
	args := m.Called(ctx, username)

	var data ytclient.ChannelData
	if args.Get(0) != nil {
		data = args.Get(0).(ytclient.ChannelData)
	}

	return data, args.Error(1)
}

func (m *MockYoutubeClient) GetChannelVideos(ctx context.Context, channelId string, maxResults int64) ([]ytclient.VideoData, error) {
	args := m.Called(ctx, channelId, maxResults)

	var data []ytclient.VideoData
	if args.Get(0) != nil {
		data = args.Get(0).([]ytclient.VideoData)
	}

	return data, args.Error(1)
}

func (m *MockYoutubeClient) GetVideosData(ctx context.Context, videoIds []string) ([]ytclient.VideoData, error) {
	args := m.Called(ctx, videoIds)

	var data []ytclient.VideoData
	if args.Get(0) != nil {
		data = args.Get(0).([]ytclient.VideoData)
	}

	return data, args.Error(1)
}

type MockWorker struct {
	mock.Mock
}

func (m *MockWorker) JobIDs(ctx context.Context) ([]int64, error) {
	args := m.Called(ctx)

	var ids []int64
	if args.Get(0) != nil {
		ids = args.Get(0).([]int64)
	}

	return ids, args.Error(1)
}

func (m *MockWorker) Execute(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
