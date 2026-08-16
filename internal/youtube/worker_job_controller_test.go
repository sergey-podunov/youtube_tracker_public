package youtube_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/youtube"
	"youtube_tracker/internal/youtube/test_utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type JobControllerResult struct {
	err      *error
	statuses []youtube.JobUpdate
}

func TestJobControllerSuite(t *testing.T) {
	suite.Run(t, new(JobControllerTestSuite))
}

type JobControllerTestSuite struct {
	suite.Suite
	jobController youtube.JobController
	mockWorker    *test_utils.MockWorker
	logger        *slog.Logger
}

func (suite *JobControllerTestSuite) SetupTest() {
	suite.mockWorker = new(test_utils.MockWorker)
	suite.logger = test_utils.NewNopLogger()

	jobController := youtube.NewWorkerJobController(suite.mockWorker)
	suite.jobController = jobController
}

func (suite *JobControllerTestSuite) TestJobController() {
	ctx := context.Background()
	ctx = context.WithValue(ctx, helpers.LoggerKey, suite.logger)
	t := suite.T()

	suite.mockWorker.On("JobIDs", mock.Anything).Return([]int64{
		int64(123456789),
	}, nil)
	suite.mockWorker.On("Execute", mock.Anything, int64(123456789)).Return(nil)

	updateChan := make(chan youtube.JobUpdate)
	jobErrorChan := make(chan *error)

	go suite.jobController.RunJob(ctx, updateChan, jobErrorChan)

	controllerResult := collectJobControllerResult(t, updateChan, jobErrorChan)
	assert.Nil(t, controllerResult.err)
	assert.Contains(t, controllerResult.statuses, youtube.JobUpdate{youtube.UpdateReady})

	suite.mockWorker.AssertExpectations(suite.T())
}

func (suite *JobControllerTestSuite) TestWorkerError() {
	ctx := suite.T().Context()
	ctx = context.WithValue(ctx, helpers.LoggerKey, suite.logger)
	t := suite.T()

	suite.mockWorker.On("JobIDs", mock.Anything).Return([]int64{
		int64(123456789),
	}, nil)
	suite.mockWorker.On("Execute", mock.Anything, int64(123456789)).Return(fmt.Errorf("worker error"))

	updateChan := make(chan youtube.JobUpdate)
	jobErrorChan := make(chan *error)

	go suite.jobController.RunJob(ctx, updateChan, jobErrorChan)

	controllerResult := collectJobControllerResult(t, updateChan, jobErrorChan)
	assert.Nil(t, controllerResult.err)
	assert.Contains(t, controllerResult.statuses, youtube.JobUpdate{youtube.UpdateError})

	suite.mockWorker.AssertExpectations(suite.T())
}

func (suite *JobControllerTestSuite) TestRunJobError() {
	ctx := suite.T().Context()
	ctx = context.WithValue(ctx, helpers.LoggerKey, suite.logger)
	t := suite.T()

	suite.mockWorker.On("JobIDs", mock.Anything).Return(nil, fmt.Errorf("repository error"))

	updateChan := make(chan youtube.JobUpdate)
	jobErrorChan := make(chan *error)

	go suite.jobController.RunJob(ctx, updateChan, jobErrorChan)

	controllerResult := collectJobControllerResult(t, updateChan, jobErrorChan)
	assert.NotNil(t, controllerResult.err)
	assert.Equal(t, *controllerResult.err, fmt.Errorf("repository error"))

	suite.mockWorker.AssertExpectations(suite.T())
}

func collectJobControllerResult(t *testing.T, updateChan chan youtube.JobUpdate, errChan chan *error) JobControllerResult {
	t.Helper()
	result := JobControllerResult{}
	timeout := time.After(5 * time.Second)

loop:
	for {
		select {
		case update, ok := <-updateChan:
			if !ok {
				break loop
			}
			result.statuses = append(result.statuses, update)
		case err, ok := <-errChan:
			if !ok {
				break loop
			}
			result.err = err
		case <-timeout:
			t.Fatal("collectJobControllerResult: timed out after 5s waiting for job channels to close")
		}
	}

	return result
}
