package service

import (
	"context"
	"errors"
	"testing"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	taskqueue "github.com/clipstudio-ai/clipstudio-ai/backend/internal/queue"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type fakeVideoRepository struct {
	video *model.Video
}

func (repo *fakeVideoRepository) Create(context.Context, *model.Video) error { return nil }
func (repo *fakeVideoRepository) GetByID(context.Context, uuid.UUID) (*model.Video, error) {
	return repo.video, nil
}
func (repo *fakeVideoRepository) Search(context.Context, string, int) ([]model.Video, error) {
	return nil, nil
}
func (repo *fakeVideoRepository) ListByUserID(context.Context, uuid.UUID, int, int) ([]model.Video, error) {
	return nil, nil
}
func (repo *fakeVideoRepository) Update(context.Context, *model.Video) error { return nil }
func (repo *fakeVideoRepository) Delete(context.Context, uuid.UUID) error    { return nil }

type fakeJobRepository struct {
	created *model.AnalysisJob
	deleted uuid.UUID
}

func (repo *fakeJobRepository) Create(_ context.Context, job *model.AnalysisJob) error {
	job.EnsureID()
	repo.created = job
	return nil
}
func (repo *fakeJobRepository) GetByID(context.Context, uuid.UUID) (*model.AnalysisJob, error) {
	return repo.created, nil
}
func (repo *fakeJobRepository) ListByVideoID(context.Context, uuid.UUID) ([]model.AnalysisJob, error) {
	return nil, nil
}
func (repo *fakeJobRepository) Update(context.Context, *model.AnalysisJob) error { return nil }
func (repo *fakeJobRepository) Delete(_ context.Context, id uuid.UUID) error {
	repo.deleted = id
	return nil
}

type fakeEnqueuer struct {
	err     error
	task    *asynq.Task
	options int
}

func (queue *fakeEnqueuer) Enqueue(task *asynq.Task, options ...asynq.Option) (*asynq.TaskInfo, error) {
	queue.task = task
	queue.options = len(options)
	return &asynq.TaskInfo{}, queue.err
}

func TestAnalyzeCreatesAndEnqueuesJob(t *testing.T) {
	videoID := uuid.New()
	jobs := &fakeJobRepository{}
	queue := &fakeEnqueuer{}
	service := NewService(
		&fakeVideoRepository{video: &model.Video{Base: model.Base{ID: videoID}}},
		jobs,
		queue,
		"analysis",
	)

	job, err := service.Analyze(context.Background(), videoID)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if job.Status != model.JobStatusQueued {
		t.Fatalf("job status = %q, want %q", job.Status, model.JobStatusQueued)
	}
	if queue.task == nil || queue.task.Type() != taskqueue.AnalysisTaskType {
		t.Fatalf("queued task = %#v, want type %q", queue.task, taskqueue.AnalysisTaskType)
	}
	if queue.options != 3 {
		t.Fatalf("queue option count = %d, want 3", queue.options)
	}
}

func TestAnalyzeRemovesJobWhenEnqueueFails(t *testing.T) {
	videoID := uuid.New()
	jobs := &fakeJobRepository{}
	service := NewService(
		&fakeVideoRepository{video: &model.Video{Base: model.Base{ID: videoID}}},
		jobs,
		&fakeEnqueuer{err: errors.New("redis unavailable")},
		"default",
	)

	job, err := service.Analyze(context.Background(), videoID)
	if err == nil {
		t.Fatal("Analyze() error = nil, want enqueue error")
	}
	if job != nil {
		t.Fatalf("Analyze() job = %#v, want nil", job)
	}
	if jobs.created == nil || jobs.deleted != jobs.created.ID {
		t.Fatalf("deleted job = %s, want created job ID", jobs.deleted)
	}
}
