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

func TestVideoWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(VideoWorkerTestSuite))
}

type VideoWorkerTestSuite struct {
	suite.Suite
	worker         *youtube.VideoWorker
	mockRepository *stats.MockVideoRepository
	mockClient     *test_utils.MockYoutubeClient
}

func (suite *VideoWorkerTestSuite) SetupTest() {
	suite.mockRepository = new(stats.MockVideoRepository)
	suite.mockClient = new(test_utils.MockYoutubeClient)
	suite.worker = youtube.NewVideoWorker(test_utils.NewNopLogger(), suite.mockRepository, suite.mockClient, 5)
}

func (suite *VideoWorkerTestSuite) TestJobIDs() {
	ctx := suite.T().Context()

	assertTime := func(before time.Time) bool {
		return before != time.Time{}
	}
	suite.mockRepository.On("GetVideos", ctx, mock.MatchedBy(assertTime), 5).Return([]stats.YoutubeVideo{
		{
			YoutubeVideoId:   int64(987654321),
			YoutubeChannelId: int64(123456789),
			ExternalID:       "vid_ext_1",
			Title:            "Sample Video",
			CreatedAt:        time.Time{},
		},
		{
			YoutubeVideoId:   int64(555444333),
			YoutubeChannelId: int64(123456789),
			ExternalID:       "vid_ext_2",
			Title:            "Another Video",
			CreatedAt:        time.Time{},
		},
	}, nil)

	ids, err := suite.worker.JobIDs(ctx)

	suite.NoError(err)
	suite.Equal([]int64{987654321, 555444333}, ids)

	suite.mockRepository.AssertExpectations(suite.T())
}

func (suite *VideoWorkerTestSuite) TestJobIDsError() {
	ctx := suite.T().Context()

	expectedError := fmt.Errorf("repository error")

	suite.mockRepository.On("GetVideos", ctx, mock.Anything, mock.Anything).Return(nil, expectedError)

	ids, err := suite.worker.JobIDs(ctx)

	suite.ErrorIs(err, expectedError)
	suite.Nil(ids)

	suite.mockRepository.AssertExpectations(suite.T())
}

func (suite *VideoWorkerTestSuite) TestExecute() {
	ctx := suite.T().Context()
	videoID := int64(987654321)

	suite.mockRepository.On("GetVideo", ctx, videoID).Return(stats.YoutubeVideo{
		YoutubeVideoId:   videoID,
		YoutubeChannelId: int64(123456789),
		ExternalID:       "vid_ext_1",
		Title:            "Sample Video",
		CreatedAt:        time.Time{},
	}, true, nil)

	suite.mockClient.On("GetVideosData", ctx, []string{"vid_ext_1"}).Return([]ytclient.VideoData{
		{
			VideoID:      "vid_ext_1",
			ChannelID:    "3263yw",
			Title:        "Sample Video",
			ViewCount:    1000,
			LikeCount:    100,
			CommentCount: 10,
		},
	}, nil)

	suite.mockRepository.On("StoreVideoStats", ctx, mock.MatchedBy(func(vs stats.YoutubeVideoStats) bool {
		return vs.YoutubeVideoID == videoID &&
			vs.ViewCount == 1000 &&
			vs.LikeCount == 100 &&
			vs.CommentCount == 10
	})).Return(stats.YoutubeVideoStats{
		YoutubeVideoStatID: int64(111),
		YoutubeVideoID:     videoID,
		ViewCount:          1000,
		LikeCount:          100,
		CommentCount:       10,
	}, nil)

	err := suite.worker.Execute(ctx, videoID)
	suite.NoError(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
}

func (suite *VideoWorkerTestSuite) TestExecuteRepositoryError() {
	ctx := suite.T().Context()
	videoID := int64(987654321)

	expectedError := fmt.Errorf("database error")

	suite.mockRepository.On("GetVideo", ctx, videoID).Return(stats.YoutubeVideo{}, false, expectedError)

	err := suite.worker.Execute(ctx, videoID)
	suite.ErrorIs(err, expectedError)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertNotCalled(suite.T(), "GetVideosData", mock.Anything, mock.Anything)
	suite.mockRepository.AssertNotCalled(suite.T(), "StoreVideoStats", mock.Anything, mock.Anything)
}

func (suite *VideoWorkerTestSuite) TestExecuteClientError() {
	ctx := suite.T().Context()
	videoID := int64(987654321)

	suite.mockRepository.On("GetVideo", ctx, videoID).Return(stats.YoutubeVideo{
		YoutubeVideoId: videoID,
		ExternalID:     "vid_ext_1",
	}, true, nil)

	suite.mockClient.On("GetVideosData", ctx, []string{"vid_ext_1"}).Return(nil, fmt.Errorf("youtube api error"))

	err := suite.worker.Execute(ctx, videoID)
	suite.NotNil(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
	suite.mockRepository.AssertNotCalled(suite.T(), "StoreVideoStats", mock.Anything, mock.Anything)
}

func (suite *VideoWorkerTestSuite) TestExecuteEmptyVideosData() {
	ctx := suite.T().Context()
	videoID := int64(987654321)

	suite.mockRepository.On("GetVideo", ctx, videoID).Return(stats.YoutubeVideo{
		YoutubeVideoId: videoID,
		ExternalID:     "vid_ext_1",
	}, true, nil)

	suite.mockClient.On("GetVideosData", ctx, []string{"vid_ext_1"}).Return([]ytclient.VideoData{}, nil)

	err := suite.worker.Execute(ctx, videoID)
	suite.NoError(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
	suite.mockRepository.AssertNotCalled(suite.T(), "StoreVideoStats", mock.Anything, mock.Anything)
}

func (suite *VideoWorkerTestSuite) TestExecuteStoreError() {
	ctx := suite.T().Context()
	videoID := int64(987654321)

	suite.mockRepository.On("GetVideo", ctx, videoID).Return(stats.YoutubeVideo{
		YoutubeVideoId: videoID,
		ExternalID:     "vid_ext_1",
	}, true, nil)

	suite.mockClient.On("GetVideosData", ctx, []string{"vid_ext_1"}).Return([]ytclient.VideoData{
		{
			VideoID:      "vid_ext_1",
			ViewCount:    1000,
			LikeCount:    100,
			CommentCount: 10,
		},
	}, nil)

	suite.mockRepository.On("StoreVideoStats", ctx, mock.Anything).Return(stats.YoutubeVideoStats{}, fmt.Errorf("store error"))

	err := suite.worker.Execute(ctx, videoID)
	suite.NotNil(err)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockClient.AssertExpectations(suite.T())
}
