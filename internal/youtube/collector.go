package youtube

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"youtube_tracker/internal/helpers"
	"youtube_tracker/internal/youtube/stats"
)

type StatisticsCollector interface {
	CollectStatistics(ctx context.Context) int64
	GetJobStatus(jobID int64) (JobView, error)
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

type job struct {
	ID         int64
	Status     JobStatus
	Total      int32
	Ready      int32
	Error      int32
	ChannelIDs []int64
	Mu         sync.RWMutex
}

type result struct {
	channelID int64
	err       error
}

type WorkerCollector struct {
	logger              *slog.Logger
	channelRepo         stats.ChannelRepository
	workers             []Worker
	jobs                map[int64]*job
	mu                  sync.Mutex
	nextJobID           int64
	channelCollectLimit int
}

const statisticsCollectorComponentName = "StatisticsCollector"

func NewStatisticsCollector(logger *slog.Logger, channelRepo stats.ChannelRepository, workers []Worker, channelLimit int) *WorkerCollector {
	return &WorkerCollector{
		logger:              logger.With(slog.String("component", statisticsCollectorComponentName)),
		channelRepo:         channelRepo,
		workers:             workers,
		jobs:                make(map[int64]*job),
		nextJobID:           1,
		channelCollectLimit: channelLimit,
	}
}

func (s *WorkerCollector) CollectStatistics(ctx context.Context) int64 {
	logger := helpers.LoggerFromContext(ctx, statisticsCollectorComponentName, s.logger)

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
	go s.runJob(ctxBackground, job)

	return jobID
}

func (s *WorkerCollector) runJob(ctx context.Context, job *job) {
	logger := helpers.LoggerFromContext(ctx, statisticsCollectorComponentName, s.logger)

	channels, err := s.channelRepo.GetChannels(ctx, time.Now().UTC(), s.channelCollectLimit)
	if err != nil {
		logger.Error("Error getting channels for job", slog.Int64("job_id", job.ID), "err", err)
		job.Status = StatusError
		return
	}

	channelsCount := len(channels)

	job.Mu.Lock()
	job.Status = StatusRunning
	job.Total = int32(channelsCount)
	logger.Info("job started", slog.Int64("job_id", job.ID), slog.Int("channels_count", channelsCount))
	job.Mu.Unlock()

	channelIDs := make([]int64, channelsCount)
	for i, ch := range channels {
		channelIDs[i] = ch.YoutubeChannelId
	}

	jobsChan := make(chan int64, len(channelIDs))
	resultChan := make(chan result, len(channelIDs))
	completeChan := make(chan bool, 1)
	
	backgroundCtx := helpers.CreateBackgroundContext(ctx, logger)
	
	for i := 0; i < len(s.workers); i++ {
		go func() {
			for channelID := range jobsChan {
				logger.Info("processing channel", slog.Int64("job_id", job.ID), slog.Int64("channel_id", channelID))
				err := s.workers[i].GetChannelStats(backgroundCtx, channelID)
				
				if err != nil {
					logger.Error("Error processing channel", slog.Int64("job_id", job.ID), slog.Int64("channel_id", channelID), "err", err)
				}
				
				res := result{channelID: channelID, err: err}
				logger.Info("channel processed",
					slog.Int64("job_id", job.ID), slog.Int64("channel_id", channelID), slog.Any("result", res))
				
				resultChan <- res
			}
		}()
	}
	
	go func() {
		for result := range resultChan {
			job.Mu.Lock()
			logger.Info("got result for channel",
				slog.Int64("job_id", job.ID), slog.Int64("channel_id", result.channelID), slog.Any("result", result))
			if result.err != nil {
				job.Error++
			} else {
				job.Ready++
			}
			
			if job.Ready+job.Error == job.Total {
				logger.Info("job completed", slog.Int64("job_id", job.ID))
				job.Status = StatusComplete
				
				completeChan <- true
			}
			
			job.Mu.Unlock()
		}
	}()
	
	for _, channelID := range channelIDs {
		jobsChan <- channelID
	}
	
	close(jobsChan)
	
	<-completeChan
	
	logger.Info("closing job result channel", slog.Int64("job_id", job.ID))
	close(completeChan)
}

func (s *WorkerCollector) GetJobStatus(jobID int64) (JobView, error) {
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
