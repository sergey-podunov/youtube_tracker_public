package youtube

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"youtube_tracker/internal/youtube/stats"
)

type StatisticsCollector interface {
	CollectStatistics(ctx context.Context) int64
	GetJobStatus(jobID int64) (*Job, error)
}

const numWorkers = 10

type JobStatus string

const (
	StatusNew      JobStatus = "NEW"
	StatusRunning  JobStatus = "RUNNING"
	StatusError    JobStatus = "ERROR"
	StatusComplete JobStatus = "COMPLETE"
)

type Job struct {
	ID         int64
	Status     JobStatus
	Total      int32
	Ready      int32
	Error      int32
	ChannelIDs []int64
}

type WorkerCollector struct {
	channelRepo         stats.ChannelRepository
	worker              Worker
	jobs                map[int64]*Job
	mu                  sync.Mutex
	nextJobID           int64
	channelCollectLimit int
}

func NewStatisticsCollector(channelRepo stats.ChannelRepository, worker Worker, channelLimit int) *WorkerCollector {
	return &WorkerCollector{
		channelRepo:         channelRepo,
		worker:              worker,
		jobs:                make(map[int64]*Job),
		nextJobID:           1,
		channelCollectLimit: channelLimit,
	}
}

func (s *WorkerCollector) CollectStatistics(ctx context.Context) int64 {
	s.mu.Lock()
	jobID := s.nextJobID
	s.nextJobID++

	job := &Job{
		ID:     jobID,
		Status: StatusNew,
		Total:  0,
		Ready:  0,
	}
	s.jobs[jobID] = job
	s.mu.Unlock()

	ctx = context.Background()
	go s.runJob(ctx, job)

	return jobID
}

func (s *WorkerCollector) runJob(ctx context.Context, job *Job) {
	channels, err := s.channelRepo.GetChannels(ctx, time.Now().UTC(), s.channelCollectLimit)
	if err != nil {
		log.Printf("Error getting channels for job %d: %v", job.ID, err)
		job.Status = StatusError
		return
	}

	channelsCount := len(channels)

	s.mu.Lock()
	job.Status = StatusRunning
	job.Total = int32(channelsCount)
	s.mu.Unlock()

	channelIDs := make([]int64, channelsCount)
	for i, ch := range channels {
		channelIDs[i] = ch.YoutubeChannelId
	}

	var wg sync.WaitGroup
	jobsChan := make(chan int64, len(channelIDs))

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for channelID := range jobsChan {
				err := s.worker.GetChannelStats(channelID)

				s.mu.Lock()
				if err != nil {
					log.Printf("Error processing channel %d for job %d: %v", channelID, job.ID, err)
					job.Error++
				} else {
					job.Ready++
				}
				s.mu.Unlock()
			}
		}()
	}

	for _, channelID := range channelIDs {
		jobsChan <- channelID
	}
	close(jobsChan)

	wg.Wait()
	job.Status = StatusComplete
}

func (s *WorkerCollector) GetJobStatus(jobID int64) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job with id %d not found", jobID)
	}
	return job, nil
}
