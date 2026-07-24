package queue

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const AnalysisTaskType = "video:analyze"

type Enqueuer interface {
	Enqueue(*asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error)
}

type AnalysisPayload struct {
	JobID   uuid.UUID `json:"job_id"`
	VideoID uuid.UUID `json:"video_id"`
}

func NewClient(redisAddress string) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddress})
}

func NewAnalysisTask(jobID, videoID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(AnalysisPayload{JobID: jobID, VideoID: videoID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(AnalysisTaskType, payload), nil
}
