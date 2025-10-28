package youtube_test

import (
	"context"
	"errors"
	"time"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"

	"github.com/stretchr/testify/mock"
)

var errUnimplemented = errors.New("unimplemented")

type EmptyChannelRepository struct{}

func (r *EmptyChannelRepository) CreateChannel(ctx context.Context, channel stats.YoutubeChannel) (*stats.YoutubeChannel, error) {
	return nil, errUnimplemented
}

func (r *EmptyChannelRepository) GetChannel(ctx context.Context, youtubeChannelId int64) (*stats.YoutubeChannel, error) {
	return nil, errUnimplemented
}

func (r *EmptyChannelRepository) StoreSubscriptionsCount(ctx context.Context, stats stats.YoutubeChannelStats) (*stats.YoutubeChannelStats, error) {
	return nil, errUnimplemented
}

func (r *EmptyChannelRepository) GetChannels(ctx context.Context, sinceTime time.Time, count int) ([]stats.YoutubeChannel, error) {
	return nil, errUnimplemented
}

// MockChannelRepository is a mock implementation of the ChannelRepository interface.
type MockChannelRepository struct {
	EmptyChannelRepository
	mock.Mock
}

func (r *MockChannelRepository) GetChannel(ctx context.Context, youtubeChannelId int64) (*stats.YoutubeChannel, error) {
	args := r.Called(ctx, youtubeChannelId)

	var ch *stats.YoutubeChannel
	if args.Get(0) != nil {
		ch = args.Get(0).(*stats.YoutubeChannel)
	}

	return ch, args.Error(1)
}

func (r *MockChannelRepository) StoreSubscriptionsCount(ctx context.Context, channelStats stats.YoutubeChannelStats) (*stats.YoutubeChannelStats, error) {
	args := r.Called(ctx, channelStats)

	var cs *stats.YoutubeChannelStats
	if args.Get(0) != nil {
		cs = args.Get(0).(*stats.YoutubeChannelStats)
	}

	return cs, args.Error(1)
}

func (r *MockChannelRepository) GetChannels(ctx context.Context, checkedBefore time.Time, count int) ([]stats.YoutubeChannel, error) {
	args := r.Called(ctx, checkedBefore, count)

	var channels []stats.YoutubeChannel
	if args.Get(0) != nil {
		channels = args.Get(0).([]stats.YoutubeChannel)
	}

	return channels, args.Error(1)
}

// MockYoutubeClient is a mock implementation of the YoutubeClient interface.
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
