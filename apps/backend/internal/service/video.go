package service

import (
	"context"
	"fmt"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	taskqueue "github.com/clipstudio-ai/clipstudio-ai/backend/internal/queue"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const searchLimit = 50

type Service struct {
	videos repository.VideoRepositoryContract
	jobs   repository.AnalysisJobRepositoryContract
	queue  taskqueue.Enqueuer
	name   string
}

func NewService(
	videos repository.VideoRepositoryContract,
	jobs repository.AnalysisJobRepositoryContract,
	queue taskqueue.Enqueuer,
	queueName string,
) *Service {
	return &Service{videos: videos, jobs: jobs, queue: queue, name: queueName}
}

func (service *Service) Search(
	ctx context.Context,
	userID uuid.UUID,
	keyword string,
) ([]model.Video, error) {
	return service.videos.Search(ctx, userID, keyword, searchLimit)
}

func (service *Service) Get(
	ctx context.Context,
	userID uuid.UUID,
	id uuid.UUID,
) (*model.Video, error) {
	return service.videos.GetByIDForUser(ctx, id, userID)
}

func (service *Service) Analyze(
	ctx context.Context,
	userID uuid.UUID,
	videoID uuid.UUID,
) (*model.AnalysisJob, error) {
	if _, err := service.videos.GetByIDForUser(ctx, videoID, userID); err != nil {
		return nil, err
	}

	job := &model.AnalysisJob{
		VideoID: videoID,
		Status:  model.JobStatusQueued,
		Message: "queued",
	}
	if err := service.jobs.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create analysis job: %w", err)
	}

	task, err := taskqueue.NewAnalysisTask(job.ID, videoID)
	if err != nil {
		_ = service.jobs.Delete(ctx, job.ID)
		return nil, fmt.Errorf("encode analysis task: %w", err)
	}

	if _, err := service.queue.Enqueue(
		task,
		asynq.Queue(service.name),
		asynq.TaskID(job.ID.String()),
		asynq.MaxRetry(3),
	); err != nil {
		_ = service.jobs.Delete(ctx, job.ID)
		return nil, fmt.Errorf("enqueue analysis task: %w", err)
	}

	return job, nil
}
