package sse

import (
	"errors"
	"net/http"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/middleware"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	jobs      repository.AnalysisJobRepositoryContract
	validate  *validator.Validate
	logger    *zap.Logger
	heartbeat time.Duration
}

type progressEvent struct {
	Step       string  `json:"step"`
	Percentage float64 `json:"percentage"`
}

func NewHandler(jobs repository.AnalysisJobRepositoryContract, logger *zap.Logger) *Handler {
	return &Handler{
		jobs: jobs, validate: validator.New(), logger: logger, heartbeat: time.Second,
	}
}

func (handler *Handler) JobEvents(c *gin.Context) {
	value := c.Param("id")
	if err := handler.validate.Var(value, "required,uuid4"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a valid UUID"})
		return
	}
	id, err := uuid.Parse(value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a valid UUID"})
		return
	}

	userID, _ := middleware.UserID(c)
	job, err := handler.jobs.GetByIDForUser(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		} else {
			handler.logger.Error("get analysis job", zap.Error(err), zap.String("job_id", id.String()))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	lastStatus := model.JobStatus("")
	lastProgress := float64(-1)
	send := func(current *model.AnalysisJob) {
		status := current.Status
		if status == model.JobStatusPending {
			status = model.JobStatusQueued
		}
		step := current.Message
		if step == "" {
			step = string(status)
		}
		c.SSEvent(string(status), progressEvent{Step: step, Percentage: current.Progress})
		c.Writer.Flush()
		lastStatus, lastProgress = current.Status, current.Progress
	}

	send(job)
	if terminal(job.Status) {
		return
	}

	ticker := time.NewTicker(handler.heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			current, getErr := handler.jobs.GetByIDForUser(c.Request.Context(), id, userID)
			if getErr != nil {
				handler.logger.Warn(
					"stop job event stream",
					zap.Error(getErr),
					zap.String("job_id", id.String()),
				)
				return
			}
			if current.Status != lastStatus || current.Progress != lastProgress {
				send(current)
			} else {
				_, _ = c.Writer.Write([]byte(": keep-alive\n\n"))
				c.Writer.Flush()
			}
			if terminal(current.Status) {
				return
			}
		}
	}
}

func terminal(status model.JobStatus) bool {
	return status == model.JobStatusCompleted ||
		status == model.JobStatusFailed ||
		status == model.JobStatusCancelled
}
