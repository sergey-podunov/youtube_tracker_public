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

func TestVideoDiscoveryWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(VideoDiscoveryWorkerTestSuite))
}

type VideoDiscoveryWorkerTestSuite struct {
	suite.Suite
	worker           *youtube.VideoDiscoveryWorker
	mockChannelRepo  *stats.MockChannelRepository
	mockVideoService *stats.MockVideoService
	mockClient       *test_utils.MockYoutubeClient
}

func (suite *VideoDiscoveryWorkerTestSuite) SetupTest() {
	suite.mockChannelRepo = new(stats.MockChannelRepository)
	suite.mockVideoService = new(stats.MockVideoService)
	suite.mockClient = new(test_utils.MockYoutubeClient)
	suite.worker = youtube.NewVideoDiscoveryWorker(test_utils.NewNopLogger(), suite.mockChannelRepo, suite.mockVideoService, suite.mockClient, 50, 5)
}

func (suite *VideoDiscoveryWorkerTestSuite) TestJobIDs() {
	ctx := suite.T().Context()

	assertTime := func(before time.Time) bool {
		return before != time.Time{}
	}
	suite.mockChannelRepo.On("GetChannels", ctx, mock.MatchedBy(assertTime), 5).Return([]stats.YoutubeChannel{
		{
			YoutubeChannelId: int64(123456789),
			ExternalID:       "UCchannel1",
			Title:            "Sample Channel",
			CreatedAt:        time.Time{},
		},
		{
			YoutubeChannelId: int64(987654321),
			ExternalID:       "UCchannel2",
			Title:            "Another Channel",
			CreatedAt:        time.Time{},
		},
	}, nil)

	ids, err := suite.worker.JobIDs(ctx)

	suite.NoError(err)
	suite.Equal([]int64{123456789, 987654321}, ids)

	suite.mockChannelRepo.AssertExpectations(suite.T())
}

func (suite *VideoDiscoveryWorkerTestSuite) TestJobIDsError() {
	ctx := suite.T().Context()

	expectedError := fmt.Errorf("repository error")

	suite.mockChannelRepo.On("GetChannels", ctx, mock.Anything, mock.Anything).Return(nil, expectedError)

	ids, err := suite.worker.JobIDs(ctx)

	suite.ErrorIs(err, expectedError)
	suite.Nil(ids)

	suite.mockChannelRepo.AssertExpectations(suite.T())
}

func (suite *VideoDiscoveryWorkerTestSuite) TestExecute() {
	ctx := suite.T().Context()
	channelID := int64(123456789)

	suite.mockChannelRepo.On("GetChannel", ctx, channelID).Return(stats.YoutubeChannel{
		YoutubeChannelId: channelID,
		ExternalID:       "UCchannel1",
		Title:            "Sample Channel",
	}, true, nil)

	suite.mockClient.On("GetChannelVideos", ctx, "UCchannel1", int64(50)).Return([]ytclient.VideoData{
		{
			VideoID:     "vid_ext_1",
			ChannelID:   "UCchannel1",
			Title:       "First Video",
			PublishedAt: "2026-05-01T10:00:00Z",
		},
		{
			VideoID:     "vid_ext_2",
			ChannelID:   "UCchannel1",
			Title:       "Second Video",
			PublishedAt: "2026-06-15T18:30:00Z",
		},
	}, nil)

	firstPublishedAt := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	suite.mockVideoService.On("CreateVideo", ctx, mock.MatchedBy(func(v stats.YoutubeVideo) bool {
		return v.YoutubeChannelId == channelID &&
			v.ExternalID == "vid_ext_1" &&
			v.Title == "First Video" &&
			v.PublishedAt != nil &&
			v.PublishedAt.Equal(firstPublishedAt)
	})).Return(stats.YoutubeVideo{YoutubeVideoId: int64(1), ExternalID: "vid_ext_1"}, true, nil)

	secondPublishedAt := time.Date(2026, 6, 15, 18, 30, 0, 0, time.UTC)
	suite.mockVideoService.On("CreateVideo", ctx, mock.MatchedBy(func(v stats.YoutubeVideo) bool {
		return v.YoutubeChannelId == channelID &&
			v.ExternalID == "vid_ext_2" &&
			v.Title == "Second Video" &&
			v.PublishedAt != nil &&
			v.PublishedAt.Equal(secondPublishedAt)
	})).Return(stats.YoutubeVideo{YoutubeVideoId: int64(2), ExternalID: "vid_ext_2"}, true, nil)

	err := suite.worker.Execute(ctx, channelID)
	suite.NoError(err)

	suite.mockChannelRepo.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
	suite.mockVideoService.AssertExpectations(suite.T())
}

func (suite *VideoDiscoveryWorkerTestSuite) TestExecuteChannelRepoError() {
	ctx := suite.T().Context()
	channelID := int64(123456789)

	suite.mockChannelRepo.On("GetChannel", ctx, channelID).Return(stats.YoutubeChannel{}, false, fmt.Errorf("database error"))

	err := suite.worker.Execute(ctx, channelID)
	suite.NotNil(err)

	suite.mockChannelRepo.AssertExpectations(suite.T())
	suite.mockClient.AssertNotCalled(suite.T(), "GetChannelVideos", mock.Anything, mock.Anything, mock.Anything)
	suite.mockVideoService.AssertNotCalled(suite.T(), "CreateVideo", mock.Anything, mock.Anything)
}

func (suite *VideoDiscoveryWorkerTestSuite) TestExecuteChannelNotFound() {
	ctx := suite.T().Context()
	channelID := int64(123456789)

	suite.mockChannelRepo.On("GetChannel", ctx, channelID).Return(stats.YoutubeChannel{}, false, nil)

	err := suite.worker.Execute(ctx, channelID)
	suite.NoError(err)

	suite.mockChannelRepo.AssertExpectations(suite.T())
	suite.mockClient.AssertNotCalled(suite.T(), "GetChannelVideos", mock.Anything, mock.Anything, mock.Anything)
	suite.mockVideoService.AssertNotCalled(suite.T(), "CreateVideo", mock.Anything, mock.Anything)
}

func (suite *VideoDiscoveryWorkerTestSuite) TestExecuteClientError() {
	ctx := suite.T().Context()
	channelID := int64(123456789)

	suite.mockChannelRepo.On("GetChannel", ctx, channelID).Return(stats.YoutubeChannel{
		YoutubeChannelId: channelID,
		ExternalID:       "UCchannel1",
	}, true, nil)

	suite.mockClient.On("GetChannelVideos", ctx, "UCchannel1", int64(50)).Return(nil, fmt.Errorf("youtube api error"))

	err := suite.worker.Execute(ctx, channelID)
	suite.NotNil(err)

	suite.mockChannelRepo.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
	suite.mockVideoService.AssertNotCalled(suite.T(), "CreateVideo", mock.Anything, mock.Anything)
}

func (suite *VideoDiscoveryWorkerTestSuite) TestExecuteEmptyResult() {
	ctx := suite.T().Context()
	channelID := int64(123456789)

	suite.mockChannelRepo.On("GetChannel", ctx, channelID).Return(stats.YoutubeChannel{
		YoutubeChannelId: channelID,
		ExternalID:       "UCchannel1",
	}, true, nil)

	suite.mockClient.On("GetChannelVideos", ctx, "UCchannel1", int64(50)).Return([]ytclient.VideoData{}, nil)

	err := suite.worker.Execute(ctx, channelID)
	suite.NoError(err)

	suite.mockChannelRepo.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
	suite.mockVideoService.AssertNotCalled(suite.T(), "CreateVideo", mock.Anything, mock.Anything)
}

func (suite *VideoDiscoveryWorkerTestSuite) TestExecuteCreateVideoError() {
	ctx := suite.T().Context()
	channelID := int64(123456789)

	suite.mockChannelRepo.On("GetChannel", ctx, channelID).Return(stats.YoutubeChannel{
		YoutubeChannelId: channelID,
		ExternalID:       "UCchannel1",
	}, true, nil)

	suite.mockClient.On("GetChannelVideos", ctx, "UCchannel1", int64(50)).Return([]ytclient.VideoData{
		{
			VideoID:     "vid_ext_1",
			ChannelID:   "UCchannel1",
			Title:       "First Video",
			PublishedAt: "2026-05-01T10:00:00Z",
		},
	}, nil)

	suite.mockVideoService.On("CreateVideo", ctx, mock.Anything).Return(stats.YoutubeVideo{}, false, fmt.Errorf("create error"))

	err := suite.worker.Execute(ctx, channelID)
	suite.NotNil(err)

	suite.mockChannelRepo.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
	suite.mockVideoService.AssertExpectations(suite.T())
}

func (suite *VideoDiscoveryWorkerTestSuite) TestExecuteInvalidPublishedAt() {
	ctx := suite.T().Context()
	channelID := int64(123456789)

	suite.mockChannelRepo.On("GetChannel", ctx, channelID).Return(stats.YoutubeChannel{
		YoutubeChannelId: channelID,
		ExternalID:       "UCchannel1",
	}, true, nil)

	suite.mockClient.On("GetChannelVideos", ctx, "UCchannel1", int64(50)).Return([]ytclient.VideoData{
		{
			VideoID:     "vid_ext_1",
			ChannelID:   "UCchannel1",
			Title:       "First Video",
			PublishedAt: "not-a-date",
		},
	}, nil)

	suite.mockVideoService.On("CreateVideo", ctx, mock.MatchedBy(func(v stats.YoutubeVideo) bool {
		return v.ExternalID == "vid_ext_1" && v.PublishedAt == nil
	})).Return(stats.YoutubeVideo{YoutubeVideoId: int64(1), ExternalID: "vid_ext_1"}, true, nil)

	err := suite.worker.Execute(ctx, channelID)
	suite.NoError(err)

	suite.mockChannelRepo.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
	suite.mockVideoService.AssertExpectations(suite.T())
}

func (suite *VideoDiscoveryWorkerTestSuite) TestExecuteDuplicate() {
	ctx := suite.T().Context()
	channelID := int64(123456789)

	suite.mockChannelRepo.On("GetChannel", ctx, channelID).Return(stats.YoutubeChannel{
		YoutubeChannelId: channelID,
		ExternalID:       "UCchannel1",
	}, true, nil)

	suite.mockClient.On("GetChannelVideos", ctx, "UCchannel1", int64(50)).Return([]ytclient.VideoData{
		{
			VideoID:     "vid_ext_1",
			ChannelID:   "UCchannel1",
			Title:       "First Video",
			PublishedAt: "2026-05-01T10:00:00Z",
		},
	}, nil)

	suite.mockVideoService.On("CreateVideo", ctx, mock.Anything).
		Return(stats.YoutubeVideo{YoutubeVideoId: int64(1), ExternalID: "vid_ext_1"}, false, nil)

	err := suite.worker.Execute(ctx, channelID)
	suite.NoError(err)

	suite.mockChannelRepo.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
	suite.mockVideoService.AssertExpectations(suite.T())
}
