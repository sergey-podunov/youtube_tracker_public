package youtube_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type StatisticsWorkerTestSuite struct {
	suite.Suite
	worker         *youtube.StatisticsWorker
	mockRepository *MockChannelRepository
	mockClient     *MockYoutubeClient
}

func (suite *StatisticsWorkerTestSuite) SetupTest() {
	suite.mockRepository = new(MockChannelRepository)
	suite.mockClient = new(MockYoutubeClient)
	suite.worker = youtube.NewStatisticsWorker(suite.mockRepository, suite.mockClient)
}

func (suite *StatisticsWorkerTestSuite) TestGetChannel() {
	ctx := context.Background()

	suite.mockRepository.On("GetChannel", ctx, int64(123456789)).Return(&stats.YoutubeChannel{
		YoutubeChannelId: int64(123456789),
		ExternalID:       "3263yw",
		Name:             "Google Dev",
		CreatedAt:        time.Time{},
	}, nil)

	suite.mockClient.On("GetChannelData", "3263yw").Return(&youtube.ChannelData{
		ChannelID:        "3263yw",
		SubscribersCount: 63362,
	}, nil)

	suite.mockRepository.On("StoreSubscriptionsCount", ctx, mock.MatchedBy(func(channelStats stats.YoutubeChannelStats) bool {
		return channelStats.YoutubeChannelID == int64(123456789) && channelStats.SubscribersCount == 63362
	})).Return(
		&stats.YoutubeChannelStats{
			YoutubeChannelStatID: int64(111),
			YoutubeChannelID:     int64(123456789),
			SubscribersCount:     63362,
		}, nil)

	err := suite.worker.GetChannelStats(int64(123456789))
	suite.NoError(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
}

func (suite *StatisticsWorkerTestSuite) TestGetChannelStatsError() {
	ctx := context.Background()
	channelID := int64(123456789)

	expectedError := fmt.Errorf("database error")

	suite.mockRepository.On("GetChannel", ctx, mock.Anything).Return(nil, expectedError)

	err := suite.worker.GetChannelStats(channelID)
	suite.NotNil(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertNotCalled(suite.T(), "GetChannelData", mock.Anything)
}

func TestStatisticsWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(StatisticsWorkerTestSuite))
}
