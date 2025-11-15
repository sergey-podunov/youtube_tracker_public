package youtube_test

import (
	"fmt"
	"testing"
	"time"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"
	"youtube_tracker/internal/youtube/test_utils"
	
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestStatisticsWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(StatisticsWorkerTestSuite))
}

type StatisticsWorkerTestSuite struct {
	suite.Suite
	worker         *youtube.StatisticsWorker
	mockRepository *stats.MockChannelRepository
	mockClient     *test_utils.MockYoutubeClient
}

func (suite *StatisticsWorkerTestSuite) SetupTest() {
	suite.mockRepository = new(stats.MockChannelRepository)
	suite.mockClient = new(test_utils.MockYoutubeClient)
	suite.worker = youtube.NewStatisticsWorker(test_utils.NewNopLogger(), suite.mockRepository, suite.mockClient)
}

func (suite *StatisticsWorkerTestSuite) TestGetChannel() {
	ctx := suite.T().Context()

	suite.mockRepository.On("GetChannel", ctx, int64(123456789)).Return(stats.YoutubeChannel{
		YoutubeChannelId: int64(123456789),
		ExternalID:       "3263yw",
		Name:             "Google Dev",
		CreatedAt:        time.Time{},
	}, true, nil)

	suite.mockClient.On("GetChannelData", ctx, "3263yw").Return(youtube.ChannelData{
		ChannelID:        "3263yw",
		SubscribersCount: 63362,
	}, nil)

	suite.mockRepository.On("StoreSubscriptionsCount", ctx, mock.MatchedBy(func(channelStats stats.YoutubeChannelStats) bool {
		return channelStats.YoutubeChannelID == int64(123456789) && channelStats.SubscribersCount == 63362
	})).Return(
		stats.YoutubeChannelStats{
			YoutubeChannelStatID: int64(111),
			YoutubeChannelID:     int64(123456789),
			SubscribersCount:     63362,
		}, nil)

	err := suite.worker.GetChannelStats(ctx, int64(123456789))
	suite.NoError(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
}

func (suite *StatisticsWorkerTestSuite) TestGetChannelStatsError() {
	ctx := suite.T().Context()
	channelID := int64(123456789)

	expectedError := fmt.Errorf("database error")

	suite.mockRepository.On("GetChannel", ctx, mock.Anything).Return(stats.YoutubeChannel{}, false, expectedError)

	err := suite.worker.GetChannelStats(ctx, channelID)
	suite.NotNil(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertNotCalled(suite.T(), "GetChannelData", mock.Anything)
}
