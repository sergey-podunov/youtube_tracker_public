package youtube

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"youtube_tracker/internal/helpers"
)

type Collector interface {
	CollectStatistics(ctx context.Context, jobController JobController) int64
	GetJobStatus(jobID int64) (JobView, error)
}

type JobController interface {
	RunJob(ctx context.Context, updateChan chan JobUpdate, jobErrorChan chan *error)
}

type JobView struct {
	ID     int64
	Status JobStatus
	Total  int32
	Ready  int32
	Error  int32
}

type JobStatus string

const (
	StatusNew      JobStatus = "NEW"
	StatusRunning  JobStatus = "RUNNING"
	StatusError    JobStatus = "ERROR"
	StatusComplete JobStatus = "COMPLETE"
)

type UpdateStatus string

const (
	UpdateReady UpdateStatus = "READY"
	UpdateError UpdateStatus = "ERROR"
)

type job struct {
	ID         int64
	Status     JobStatus
	Total      int32
	Ready      int32
	Error      int32
	ChannelIDs []int64
	Mu         sync.RWMutex
}

type JobUpdate struct {
	Status UpdateStatus
}

type chanResult struct {
	id  int64
	err error
}

type StatisticsCollector struct {
	logger    *slog.Logger
	jobs      map[int64]*job
	mu        sync.Mutex
	nextJobID int64
}

const statisticsCollectorComponentName = "StatisticsCollector"

func NewStatisticsCollector(logger *slog.Logger) *StatisticsCollector {
	return &StatisticsCollector{
		logger:    logger.With(slog.String("component", statisticsCollectorComponentName)),
		jobs:      make(map[int64]*job),
		nextJobID: 1,
	}
}

func (s *StatisticsCollector) CollectStatistics(ctx context.Context, jobController JobController) int64 {
	logger := helpers.LoggerFromContextWithDefault(ctx, statisticsCollectorComponentName, s.logger)

	s.mu.Lock()
	jobID := s.nextJobID
	s.nextJobID++

	job := &job{
		ID:     jobID,
		Status: StatusNew,
		Total:  0,
		Ready:  0,
	}
	s.jobs[jobID] = job

	logger.Info("Job created", slog.Int64("job_id", jobID))
	s.mu.Unlock()

	ctxBackground := context.WithValue(context.Background(), helpers.LoggerKey, logger)
	updateChan := make(chan JobUpdate)
	jobErrorChan := make(chan *error)

	go jobController.RunJob(ctxBackground, updateChan, jobErrorChan)
	go s.updateJobStatus(logger, jobID, updateChan, jobErrorChan)

	return jobID
}

func (s *StatisticsCollector) updateJobStatus(logger *slog.Logger, jobID int64, updateChan chan JobUpdate, errorChan chan *error) {
	s.mu.Lock()
	job, ok := s.jobs[jobID]
	s.mu.Unlock()

	if !ok {
		logger.Error("Job not found", slog.Int64("job_id", jobID))
		err := fmt.Errorf("job not found: %d", jobID)
		errorChan <- &err
	}

	var jobStatus JobStatus

channelLoop:
	for {
		select {
		case jobUpdate, ok := <-updateChan:
			if !ok {
				break channelLoop
			}

			job.Mu.Lock()
			logger.Info("got update for job", slog.Int64("job_id", jobID), slog.Any("status", jobUpdate.Status))

			switch jobUpdate.Status {
			case UpdateError:
				job.Error++
			case UpdateReady:
				job.Ready++
			}
			job.Total++

			job.Mu.Unlock()
		case err := <-errorChan:
			if err != nil {
				jobStatus = StatusError
				break channelLoop
			}
		}
	}

	logger.Info("job completed", slog.Int64("job_id", job.ID))

	job.Mu.Lock()

	if jobStatus != StatusError {
		jobStatus = StatusComplete
	}
	job.Status = jobStatus

	job.Mu.Unlock()
}

func (s *StatisticsCollector) GetJobStatus(jobID int64) (JobView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return JobView{}, fmt.Errorf("job with id %d not found", jobID)
	}

	return JobView{
		ID:     job.ID,
		Status: job.Status,
		Total:  job.Total,
		Ready:  job.Ready,
		Error:  job.Error,
	}, nil
}
