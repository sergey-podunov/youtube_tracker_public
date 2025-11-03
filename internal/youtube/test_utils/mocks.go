package test_utils

import (
	"errors"
	"youtube_tracker/internal/youtube"
	
	"github.com/stretchr/testify/mock"
)

var errUnimplemented = errors.New("unimplemented")

type MockYoutubeClient struct {
	youtube.EmptyClient
	mock.Mock
}

func (m *MockYoutubeClient) GetChannelData(channelId string) (*youtube.ChannelData, error) {
	args := m.Called(channelId)

	var data *youtube.ChannelData
	if args.Get(0) != nil {
		data = args.Get(0).(*youtube.ChannelData)
	}

	return data, args.Error(1)
}

type EmptyStatisticsWorker struct{}

func (e *EmptyStatisticsWorker) GetChannelStats(channelID int64) error {
	return errUnimplemented
}

type MockStatisticsWorker struct {
	EmptyStatisticsWorker
	mock.Mock
}

func (m *MockStatisticsWorker) GetChannelStats(channelID int64) error {
	args := m.Called(channelID)
	return args.Error(0)
}
