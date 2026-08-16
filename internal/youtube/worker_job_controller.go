package youtube

import (
	"context"
	"log/slog"
	"sync"
	"youtube_tracker/internal/helpers"
)

type Worker interface {
	JobIDs(ctx context.Context) ([]int64, error)
	Execute(ctx context.Context, id int64) error
}

type WorkerJobController struct {
	worker Worker
}

const jobControllerComponentName = "JobController"

func NewWorkerJobController(worker Worker) *WorkerJobController {
	return &WorkerJobController{
		worker: worker,
	}
}

func (c *WorkerJobController) RunJob(ctx context.Context, updateChan chan JobUpdate, jobErrorChan chan *error) {
	defer func() {
		close(updateChan)
		close(jobErrorChan)
	}()
	logger := helpers.LoggerFromContext(ctx, jobControllerComponentName)

	ids, err := c.worker.JobIDs(ctx)
	if err != nil {
		jobErrorChan <- &err
		return
	}

	jobsChan := make(chan int64, len(ids))
	resultChan := make(chan chanResult, len(ids))

	backgroundCtx := helpers.CreateBackgroundContext(ctx, logger)

	for _, id := range ids {
		jobsChan <- id
	}
	close(jobsChan)

	var wg sync.WaitGroup
	wg.Go(func() {
		for id := range jobsChan {
			logger.Info("Execute job", slog.Int64("id", id))
			err := c.worker.Execute(backgroundCtx, id)

			if err != nil {
				logger.Error("Error in job execution", slog.Int64("id", id), "err", err)
			}

			res := chanResult{id: id, err: err}
			logger.Info("Job finished", slog.Int64("id", id),
				slog.Int64("id", id), slog.Any("result", res))

			resultChan <- res
		}
	})

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for res := range resultChan {
		if res.err != nil {
			updateChan <- JobUpdate{UpdateError}
		} else {
			updateChan <- JobUpdate{UpdateReady}
		}
	}
}
