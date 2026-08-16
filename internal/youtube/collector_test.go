package youtube_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/test_utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestWorkerCollectorSuite(t *testing.T) {
	suite.Run(t, new(StatisticsCollectorTestSuite))
}

type dummyJobController struct {
	statuses []youtube.UpdateStatus
	err      *error
}

func (controller dummyJobController) RunJob(ctx context.Context, updateChan chan youtube.JobUpdate, jobErrorChan chan *error) {
	defer func() {
		close(jobErrorChan)
		close(updateChan)
	}()

	if controller.err != nil {
		jobErrorChan <- controller.err
		return
	}

	for _, status := range controller.statuses {
		updateChan <- youtube.JobUpdate{Status: status}
	}
}

type StatisticsCollectorTestSuite struct {
	suite.Suite
	collector youtube.Collector
	logger    *slog.Logger
}

func (suite *StatisticsCollectorTestSuite) SetupTest() {
	suite.logger = test_utils.NewNopLogger()

	collector := youtube.NewStatisticsCollector(suite.logger)
	suite.collector = collector
}

func (suite *StatisticsCollectorTestSuite) TestStatisticsCollector() {
	ctx := context.Background()
	t := suite.T()

	jobController := dummyJobController{
		statuses: []youtube.UpdateStatus{youtube.UpdateReady, youtube.UpdateError, youtube.UpdateReady},
	}
	jobID := suite.collector.CollectStatistics(ctx, jobController)
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
	assert.Equal(t, int32(3), actualJob.Total)
	assert.Equal(t, int32(2), actualJob.Ready)
	assert.Equal(t, int32(1), actualJob.Error)
}

func (suite *StatisticsCollectorTestSuite) TestStatisticsCollectorJobError() {
	ctx := suite.T().Context()
	t := suite.T()

	err := errors.New("some error")
	jobController := dummyJobController{
		err: &err,
	}
	jobID := suite.collector.CollectStatistics(ctx, jobController)
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
}
