package youtube_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/stats"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type StatisticsCollectorTestSuite struct {
	suite.Suite
	collector      youtube.StatisticsCollector
	mockWorker     *MockStatisticsWorker
	mockRepository *MockChannelRepository
}

func (suite *StatisticsCollectorTestSuite) SetupTest() {
	suite.mockRepository = new(MockChannelRepository)
	suite.mockWorker = new(MockStatisticsWorker)
	workers := []youtube.Worker{suite.mockWorker}

	suite.collector = youtube.NewStatisticsCollector(suite.mockRepository, workers, 5)
}

func (suite *StatisticsCollectorTestSuite) TestStatisticsCollector() {
	ctx := context.Background()
	t := suite.T()

	assertTime := func(before time.Time) bool {
		return before != time.Time{}
	}
	suite.mockRepository.On("GetChannels", ctx, mock.MatchedBy(assertTime), 5).Return([]stats.YoutubeChannel{
		{
			YoutubeChannelId: int64(123456789),
			ExternalId:       "3263yw",
			Name:             "Google Dev",
			CreatedAt:        time.Time{},
		},
	}, nil)

	suite.mockWorker.On("GetChannelStats", int64(123456789)).Return(nil)

	jobID := suite.collector.CollectStatistics(ctx)
	assert.Equal(t, int64(1), jobID)

	var actualJob youtube.JobView

	assert.Eventually(t, func() bool {
		job, err := suite.collector.GetJobStatus(jobID)
		if err != nil {
			return false
		}

		actualJob = job

		return job.Status == youtube.StatusComplete || job.Status == youtube.StatusError
	}, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, youtube.StatusComplete, actualJob.Status)
	assert.Equal(t, int32(1), actualJob.Total)
	assert.Equal(t, int32(1), actualJob.Ready)
	assert.Equal(t, int32(0), actualJob.Error)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockWorker.AssertExpectations(suite.T())
}

func (suite *StatisticsCollectorTestSuite) TestStatisticsCollectorWorkerError() {
	ctx := context.Background()
	t := suite.T()

	suite.mockRepository.On("GetChannels", ctx, mock.Anything, mock.Anything).Return([]stats.YoutubeChannel{
		{
			YoutubeChannelId: int64(123456789),
			ExternalId:       "3263yw",
			Name:             "Google Dev",
			CreatedAt:        time.Time{},
		},
	}, nil)

	suite.mockWorker.On("GetChannelStats", int64(123456789)).Return(fmt.Errorf("worker error"))

	jobID := suite.collector.CollectStatistics(ctx)
	assert.Equal(t, int64(1), jobID)

	var actualJob youtube.JobView

	assert.Eventually(t, func() bool {
		job, err := suite.collector.GetJobStatus(jobID)
		if err != nil {
			return false
		}

		actualJob = job

		return job.Status == youtube.StatusComplete || job.Status == youtube.StatusError
	}, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, youtube.StatusComplete, actualJob.Status)
	assert.Equal(t, int32(1), actualJob.Total)
	assert.Equal(t, int32(0), actualJob.Ready)
	assert.Equal(t, int32(1), actualJob.Error)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockWorker.AssertExpectations(suite.T())
}

func (suite *StatisticsCollectorTestSuite) TestStatisticsCollectorRunJobError() {
	ctx := context.Background()
	t := suite.T()

	suite.mockRepository.On("GetChannels", ctx, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("repository error"))

	jobID := suite.collector.CollectStatistics(ctx)
	assert.Equal(t, int64(1), jobID)

	var actualJob youtube.JobView

	assert.Eventually(t, func() bool {
		job, err := suite.collector.GetJobStatus(jobID)
		if err != nil {
			return false
		}

		actualJob = job

		return job.Status == youtube.StatusComplete || job.Status == youtube.StatusError
	}, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, youtube.StatusError, actualJob.Status)
	assert.Equal(t, int32(0), actualJob.Total)
	assert.Equal(t, int32(0), actualJob.Ready)
	assert.Equal(t, int32(0), actualJob.Error)

	suite.mockRepository.AssertExpectations(suite.T())
	suite.mockWorker.AssertExpectations(suite.T())
}

func TestWorkerCollectorSuite(t *testing.T) {
	suite.Run(t, new(StatisticsCollectorTestSuite))
}
