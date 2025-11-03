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
	GetJobStatus(jobID int64) (JobView, error)
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

type JobView struct {
	ID     int64
	Status JobStatus
	Total  int32
	Ready  int32
	Error  int32
}

type result struct {
	channelID int64
	err       error
}

type WorkerCollector struct {
	channelRepo         stats.ChannelRepository
	workers             []Worker
	jobs                map[int64]*job
	mu                  sync.Mutex
	nextJobID           int64
	channelCollectLimit int
}

func NewStatisticsCollector(channelRepo stats.ChannelRepository, workers []Worker, channelLimit int) WorkerCollector {
	return WorkerCollector{
		channelRepo:         channelRepo,
		workers:             workers,
		jobs:                make(map[int64]*job),
		nextJobID:           1,
		channelCollectLimit: channelLimit,
	}
}

func (s *WorkerCollector) CollectStatistics(ctx context.Context) int64 {
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
	s.mu.Unlock()

	ctxBackground := context.Background()
	go s.runJob(ctxBackground, job)

	return jobID
}

func (s *WorkerCollector) runJob(ctx context.Context, job *job) {
	channels, err := s.channelRepo.GetChannels(ctx, time.Now().UTC(), s.channelCollectLimit)
	if err != nil {
		log.Printf("Error getting channels for job %d: %v", job.ID, err)
		job.Status = StatusError

		return
	}

	channelsCount := len(channels)

	job.Mu.Lock()
	job.Status = StatusRunning
	job.Total = int32(channelsCount)
	job.Mu.Unlock()

	channelIDs := make([]int64, channelsCount)
	for i, ch := range channels {
		channelIDs[i] = ch.YoutubeChannelId
	}

	jobsChan := make(chan int64, len(channelIDs))
	resultChan := make(chan result, len(channelIDs))
	completeChan := make(chan bool, 1)

	for i := 0; i < len(s.workers); i++ {
		go func() {
			for channelID := range jobsChan {
				log.Printf("processing channel %d for job %d", channelID, job.ID)
				err := s.workers[i].GetChannelStats(channelID)

				if err != nil {
					log.Printf("Error processing channel %d for job %d: %#v", channelID, job.ID, err)
				}

				res := result{channelID: channelID, err: err}
				log.Printf("sending result of channel %d for job %d: %#v", channelID, job.ID, res)

				resultChan <- res
			}
		}()
	}

	go func() {
		for result := range resultChan {
			job.Mu.Lock()
			log.Printf("got result for channel %d, job %d: %#v", result.channelID, job.ID, result)
			if result.err != nil {
				job.Error++
			} else {
				job.Ready++
			}

			if job.Ready+job.Error == job.Total {
				log.Printf("job %d is complete", job.ID)
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

	log.Printf("closing job %d result channel", job.ID)
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
		Error: job.Error,
	}, nil
}
