package youtube

import (
	"context"
	"fmt"
	"testing"
	"time"
	"youtube_tracker/internal/youtube/stats"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MockChannelRepository struct {
	stats.EmptyChannelRepository
	mock.Mock
}

func (r *MockChannelRepository) GetChannel(ctx context.Context, youtubeChannelId int64) (*stats.YoutubeChannel, error) {
	args := r.Called(ctx, youtubeChannelId)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*stats.YoutubeChannel), args.Error(1)
}

func (r *MockChannelRepository) StoreSubscriptionsCount(ctx context.Context, channelStats stats.YoutubeChannelStats) (*stats.YoutubeChannelStats, error) {
	args := r.Called(ctx, channelStats)
	return args.Get(0).(*stats.YoutubeChannelStats), args.Error(1)
}

type MockYoutubeClient struct {
	EmptyClient
	mock.Mock
}

func (m *MockYoutubeClient) GetChannelData(channelId string) (*ChannelData, error) {
	args := m.Called(channelId)
	return args.Get(0).(*ChannelData), args.Error(1)
}

type YoutubeChannelWorkerTestSuite struct {
	suite.Suite
	worker         *YoutubeChannelWorker
	mockRepository *MockChannelRepository
	mockClient     *MockYoutubeClient
}

func (suite *YoutubeChannelWorkerTestSuite) SetupTest() {

	suite.mockRepository = new(MockChannelRepository)
	suite.mockClient = new(MockYoutubeClient)

	suite.worker = &YoutubeChannelWorker{
		channelRep: suite.mockRepository,
		client:     suite.mockClient,
	}
}

func (suite *YoutubeChannelWorkerTestSuite) TestGetChannel() {
	ctx := context.Background()

	suite.mockRepository.On("GetChannel", ctx, int64(123456789)).Return(&stats.YoutubeChannel{
		YoutubeChannelId: int64(123456789),
		ExternalId:       "3263yw",
		Name:             "Google Dev",
		CreatedAt:        time.Time{},
	}, nil)

	suite.mockClient.On("GetChannelData", "3263yw").Return(&ChannelData{
		ChannelID:        "3263yw",
		SubscribersCount: 63362,
	}, nil)

	suite.mockRepository.On("StoreSubscriptionsCount", ctx, mock.MatchedBy(func(channelStats stats.YoutubeChannelStats) bool {
		return channelStats.YoutubeChannelId == int64(123456789) && channelStats.SubscribersCount == 63362
	})).Return(
		&stats.YoutubeChannelStats{
			YoutubeChannelStatId: int64(111),
			YoutubeChannelId:     int64(123456789),
			SubscribersCount:     63362,
		}, nil)

	err := suite.worker.GetChannelStats(int64(123456789))
	suite.NoError(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
}

func (suite *YoutubeChannelWorkerTestSuite) TestGetChannelStatsError() {
	ctx := context.Background()
	channelID := int64(123456789)

	expectedError := fmt.Errorf("database error")

	suite.mockRepository.On("GetChannel", ctx, mock.Anything).Return(nil, expectedError)

	err := suite.worker.GetChannelStats(channelID)
	suite.NotNil(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertNotCalled(suite.T(), "GetChannelData", mock.Anything)
}

func TestYoutubeChannelWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(YoutubeChannelWorkerTestSuite))
}
