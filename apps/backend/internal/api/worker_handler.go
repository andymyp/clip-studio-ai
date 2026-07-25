package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/middleware"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type WorkerHandler struct {
	worker   *service.WorkerClient
	validate *validator.Validate
	logger   *zap.Logger
}

type analyzeDiscoveredVideoRequest struct {
	ExternalID string `json:"external_id" validate:"required,max=128"`
	URL        string `json:"url" validate:"required,url,max=2048"`
	Title      string `json:"title" validate:"required,max=500"`
	Platform   string `json:"platform" validate:"required,oneof=youtube"`
	Thumbnail  string `json:"thumbnail" validate:"omitempty,url,max=2048"`
}

func NewWorkerHandler(worker *service.WorkerClient, logger *zap.Logger) *WorkerHandler {
	return &WorkerHandler{worker: worker, validate: validator.New(), logger: logger}
}

func (handler *WorkerHandler) CreateAnalysis(c *gin.Context) {
	var request analyzeDiscoveredVideoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	request.URL = strings.TrimSpace(request.URL)
	request.Title = strings.TrimSpace(request.Title)
	request.ExternalID = strings.TrimSpace(request.ExternalID)
	if err := handler.validate.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "a valid YouTube URL, title, and platform are required",
		})
		return
	}
	userID, _ := middleware.UserID(c)
	job, err := handler.worker.CreateAnalysis(c.Request.Context(), service.WorkerAnalyzeRequest{
		UserID: userID.String(), ExternalID: request.ExternalID,
		URL: request.URL, Title: request.Title,
		Platform: request.Platform, Thumbnail: request.Thumbnail,
	})
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	c.JSON(http.StatusAccepted, job)
}

func (handler *WorkerHandler) GetReview(c *gin.Context) {
	job, ok := handler.reviewForUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, job)
}

func (handler *WorkerHandler) GetAnalysis(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if err := handler.validate.Var(id, "required,uuid4"); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "id must be a valid UUID"})
		return
	}
	job, err := handler.worker.GetAnalysis(c.Request.Context(), id)
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	userID, _ := middleware.UserID(c)
	if job.UserID != userID.String() {
		c.JSON(http.StatusNotFound, errorResponse{Error: "resource not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

func (handler *WorkerHandler) AnalysisEvents(c *gin.Context) {
	job, ok := handler.reviewForUser(c)
	if !ok {
		return
	}
	externalID := strings.TrimSpace(c.Param("external_id"))
	userID, _ := middleware.UserID(c)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	send := func(current *service.WorkerAnalysisJob) {
		c.SSEvent(current.Status, current)
		c.Writer.Flush()
	}
	send(job)
	if workerJobTerminal(job.Status) {
		return
	}

	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	lastStatus, lastProgress, lastMessage := job.Status, job.Progress, job.Message
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			current, getErr := handler.worker.GetReview(
				c.Request.Context(), userID.String(), externalID,
			)
			if getErr != nil {
				handler.logger.Warn("worker SSE stream stopped", zap.Error(getErr), zap.String("external_id", externalID))
				c.SSEvent("error", gin.H{"error": "worker connection lost"})
				c.Writer.Flush()
				return
			}
			if current.Status != lastStatus ||
				current.Progress != lastProgress ||
				current.Message != lastMessage {
				send(current)
				lastStatus, lastProgress, lastMessage = current.Status, current.Progress, current.Message
			} else {
				_, _ = c.Writer.Write([]byte(": keep-alive\n\n"))
				c.Writer.Flush()
			}
			if workerJobTerminal(current.Status) {
				return
			}
		}
	}
}

func (handler *WorkerHandler) reviewForUser(c *gin.Context) (*service.WorkerAnalysisJob, bool) {
	externalID := strings.TrimSpace(c.Param("external_id"))
	if err := handler.validate.Var(externalID, "required,max=128"); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "external_id is required"})
		return nil, false
	}
	userID, _ := middleware.UserID(c)
	job, err := handler.worker.GetReview(c.Request.Context(), userID.String(), externalID)
	if err != nil {
		writeError(c, handler.logger, err)
		return nil, false
	}
	if job.UserID != userID.String() {
		c.JSON(http.StatusNotFound, errorResponse{Error: "resource not found"})
		return nil, false
	}
	return job, true
}

func workerJobTerminal(status string) bool {
	return status == "completed" || status == "failed"
}
