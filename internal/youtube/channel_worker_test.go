package youtube_test

import (
	"fmt"
	"testing"
	"time"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"
	"youtube_tracker/internal/youtube/test_utils"
	"youtube_tracker/internal/ytclient"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestSChannelWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(ChannelWorkerTestSuite))
}

type ChannelWorkerTestSuite struct {
	suite.Suite
	worker         *youtube.ChannelWorker
	mockRepository *stats.MockChannelRepository
	mockClient     *test_utils.MockYoutubeClient
}

func (suite *ChannelWorkerTestSuite) SetupTest() {
	suite.mockRepository = new(stats.MockChannelRepository)
	suite.mockClient = new(test_utils.MockYoutubeClient)
	suite.worker = youtube.NewChannelWorker(test_utils.NewNopLogger(), suite.mockRepository, suite.mockClient, 5)
}

func (suite *ChannelWorkerTestSuite) TestJobIDs() {
	ctx := suite.T().Context()

	assertTime := func(before time.Time) bool {
		return before != time.Time{}
	}
	suite.mockRepository.On("GetChannels", ctx, mock.MatchedBy(assertTime), 5).Return([]stats.YoutubeChannel{
		{
			YoutubeChannelId: int64(123456789),
			ExternalID:       "3263yw",
			Title:            "Google Dev",
			CreatedAt:        time.Time{},
		},
		{
			YoutubeChannelId: int64(987654321),
			ExternalID:       "8fj2kd",
			Title:            "Go Time",
			CreatedAt:        time.Time{},
		},
	}, nil)

	ids, err := suite.worker.JobIDs(ctx)

	suite.NoError(err)
	suite.Equal([]int64{123456789, 987654321}, ids)

	suite.mockRepository.AssertExpectations(suite.T())
}

func (suite *ChannelWorkerTestSuite) TestJobIDsError() {
	ctx := suite.T().Context()

	expectedError := fmt.Errorf("repository error")

	suite.mockRepository.On("GetChannels", ctx, mock.Anything, mock.Anything).Return(nil, expectedError)

	ids, err := suite.worker.JobIDs(ctx)

	suite.ErrorIs(err, expectedError)
	suite.Nil(ids)

	suite.mockRepository.AssertExpectations(suite.T())
}

func (suite *ChannelWorkerTestSuite) TestExecute() {
	ctx := suite.T().Context()

	suite.mockRepository.On("GetChannel", ctx, int64(123456789)).Return(stats.YoutubeChannel{
		YoutubeChannelId: int64(123456789),
		ExternalID:       "3263yw",
		Title:            "Google Dev",
		CreatedAt:        time.Time{},
	}, true, nil)

	suite.mockClient.On("GetChannelData", ctx, "3263yw").Return(ytclient.ChannelData{
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

	err := suite.worker.Execute(ctx, int64(123456789))
	suite.NoError(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
}

func (suite *ChannelWorkerTestSuite) TestExecuteError() {
	ctx := suite.T().Context()

	expectedError := fmt.Errorf("worker error")

	suite.mockRepository.On("GetChannel", ctx, int64(123456789)).Return(stats.YoutubeChannel{}, false, expectedError)

	err := suite.worker.Execute(ctx, int64(123456789))
	suite.ErrorIs(err, expectedError)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertNotCalled(suite.T(), "GetChannelData", mock.Anything)
}
