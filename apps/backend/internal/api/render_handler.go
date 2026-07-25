package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/middleware"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RenderHandler struct {
	db     *gorm.DB
	redis  *redis.Client
	worker *service.WorkerClient
	logger *zap.Logger
}

type createRendersRequest struct {
	ExternalID    string   `json:"external_id" binding:"required"`
	ClipIDs       []string `json:"clip_ids" binding:"required,min=1,max=8"`
	WatermarkText string   `json:"watermark_text" binding:"max=100"`
}

type renderJobResponse struct {
	ID                uuid.UUID       `json:"id"`
	ClipID            uuid.UUID       `json:"clip_id"`
	ExternalID        string          `json:"external_id"`
	VideoTitle        string          `json:"video_title"`
	Thumbnail         string          `json:"thumbnail"`
	Status            model.JobStatus `json:"status"`
	Progress          float64         `json:"progress"`
	Message           string          `json:"message"`
	Score             float64         `json:"score"`
	Duration          float64         `json:"duration"`
	OutputPath        string          `json:"output_path"`
	MediaURL          string          `json:"media_url"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	Hashtags          []string        `json:"hashtags"`
	Error             string          `json:"error"`
	Hook              string          `json:"hook"`
	RemovedSeconds    float64         `json:"removed_seconds"`
	PatternInterrupts int             `json:"pattern_interrupts"`
	SourceUsername    string          `json:"source_username"`
	PlaybackSpeed     float64         `json:"playback_speed"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type createFeedbackRequest struct {
	Platform      string  `json:"platform" binding:"required,oneof=youtube tiktok instagram"`
	Views         int64   `json:"views" binding:"min=0"`
	Likes         int64   `json:"likes" binding:"min=0"`
	Comments      int64   `json:"comments" binding:"min=0"`
	Shares        int64   `json:"shares" binding:"min=0"`
	AverageWatch  float64 `json:"average_watch_seconds" binding:"min=0"`
	CompletionPct float64 `json:"completion_percentage" binding:"min=0,max=100"`
}

func NewRenderHandler(
	db *gorm.DB,
	redisClient *redis.Client,
	worker *service.WorkerClient,
	logger *zap.Logger,
) *RenderHandler {
	return &RenderHandler{db: db, redis: redisClient, worker: worker, logger: logger}
}

func (handler *RenderHandler) Create(c *gin.Context) {
	var body createRendersRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "external_id and clip_ids are required"})
		return
	}
	userID, _ := middleware.UserID(c)
	review, err := handler.worker.GetReview(c.Request.Context(), userID.String(), body.ExternalID)
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	if review.Status != "completed" {
		c.JSON(http.StatusConflict, errorResponse{Error: "clip analysis must complete before rendering"})
		return
	}

	video, err := handler.findOrCreateVideo(c, userID, review)
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	available := make(map[string]service.WorkerClip, len(review.Clips))
	for _, clip := range review.Clips {
		available[clip.ID] = clip
	}

	responses := make([]renderJobResponse, 0, len(body.ClipIDs))
	for _, value := range body.ClipIDs {
		source, found := available[value]
		clipID, parseErr := uuid.Parse(value)
		if !found || parseErr != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "one or more clip_ids are invalid"})
			return
		}
		clip := model.Clip{
			Base: model.Base{ID: clipID}, VideoID: video.ID,
			StartTime: source.Start, EndTime: source.End, Score: source.Score,
			Reason: source.Reason, Status: model.ClipStatusApproved,
			SourcePath: source.OutputPath,
		}
		if err = handler.db.WithContext(c.Request.Context()).Where("id = ?", clip.ID).
			Assign(clip).FirstOrCreate(&clip).Error; err != nil {
			writeError(c, handler.logger, err)
			return
		}
		job := model.RenderJob{
			ClipID: clip.ID, Status: model.JobStatusQueued,
			Message: "Queued for rendering", WatermarkText: strings.TrimSpace(body.WatermarkText),
			SourceURL: review.URL, SourceUsername: review.YouTubeUsername,
		}
		job.EnsureID()
		if err = handler.db.WithContext(c.Request.Context()).Create(&job).Error; err != nil {
			writeError(c, handler.logger, err)
			return
		}
		_, workerErr := handler.worker.CreateRender(c.Request.Context(), service.WorkerRenderRequest{
			ID: job.ID.String(), UserID: userID.String(), AnalysisJobID: review.ID,
			ClipID: clip.ID.String(), ClipPath: clip.SourcePath,
			Start: clip.StartTime, End: clip.EndTime,
			WatermarkText: job.WatermarkText, SourceURL: job.SourceURL,
			SourceUsername: job.SourceUsername,
		})
		if workerErr != nil {
			job.Status, job.Message, job.Error = model.JobStatusFailed, "Could not queue render", workerErr.Error()
			_ = handler.db.WithContext(c.Request.Context()).Save(&job).Error
		}
		job.Clip, job.Clip.Video = clip, *video
		responses = append(responses, renderResponse(job))
	}
	c.JSON(http.StatusAccepted, responses)
}

func (handler *RenderHandler) List(c *gin.Context) {
	handler.list(c, false)
}

func (handler *RenderHandler) Rendered(c *gin.Context) {
	handler.list(c, true)
}

func (handler *RenderHandler) Events(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	lastPayload := ""
	send := func() bool {
		jobs, err := handler.listOwned(c, userID)
		if err != nil {
			handler.logger.Warn("render SSE query failed", zap.Error(err))
			return false
		}
		responses := make([]renderJobResponse, 0, len(jobs))
		for index := range jobs {
			handler.sync(c, &jobs[index])
			responses = append(responses, renderResponse(jobs[index]))
		}
		payload, err := json.Marshal(responses)
		if err != nil {
			return false
		}
		current := string(payload)
		if current == lastPayload {
			_, _ = c.Writer.Write([]byte(": keep-alive\n\n"))
		} else {
			c.SSEvent("renders", responses)
			lastPayload = current
		}
		c.Writer.Flush()
		return true
	}
	if !send() {
		c.SSEvent("error", gin.H{"error": "could not load render jobs"})
		return
	}
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			if !send() {
				c.SSEvent("error", gin.H{"error": "render event stream stopped"})
				c.Writer.Flush()
				return
			}
		}
	}
}

func (handler *RenderHandler) list(c *gin.Context, completedOnly bool) {
	userID, _ := middleware.UserID(c)
	jobs, err := handler.listOwned(c, userID)
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	responses := make([]renderJobResponse, 0, len(jobs))
	for index := range jobs {
		handler.sync(c, &jobs[index])
		if completedOnly && jobs[index].Status != model.JobStatusCompleted {
			continue
		}
		responses = append(responses, renderResponse(jobs[index]))
	}
	c.JSON(http.StatusOK, responses)
}

func (handler *RenderHandler) Retry(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid render job id"})
		return
	}
	job, err := handler.getOwned(c, userID, id)
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	if job.Status != model.JobStatusFailed {
		c.JSON(http.StatusConflict, errorResponse{Error: "only failed jobs can be retried"})
		return
	}
	state, err := handler.worker.RetryRender(c.Request.Context(), id.String())
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	applyRenderState(job, state)
	_ = handler.db.WithContext(c.Request.Context()).Save(job).Error
	if job.Status == model.JobStatusCompleted {
		_ = handler.db.WithContext(c.Request.Context()).Model(&model.Clip{}).
			Where("id = ?", job.ClipID).Update("status", model.ClipStatusRendered).Error
	}
	c.JSON(http.StatusAccepted, renderResponse(*job))
}

func (handler *RenderHandler) CreateFeedback(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid render job id"})
		return
	}
	job, err := handler.getOwned(c, userID, id)
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	if job.Status != model.JobStatusCompleted {
		c.JSON(http.StatusConflict, errorResponse{Error: "feedback requires a completed render"})
		return
	}
	var body createFeedbackRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid performance metrics"})
		return
	}
	engagement := float64(body.Likes+body.Comments*2+body.Shares*4) /
		float64(max(body.Views, 1)) * 100
	viralScore := min(100, body.CompletionPct*0.55+min(engagement, 100)*0.35+
		min(body.AverageWatch/max(job.Clip.EndTime-job.Clip.StartTime, 1)*100, 100)*0.10)
	feedback := model.ClipPerformance{
		RenderJobID: id, Platform: body.Platform, Views: body.Views, Likes: body.Likes,
		Comments: body.Comments, Shares: body.Shares, AverageWatch: body.AverageWatch,
		CompletionPct: body.CompletionPct, ViralScore: viralScore,
	}
	feedback.EnsureID()
	if err = handler.db.WithContext(c.Request.Context()).Create(&feedback).Error; err != nil {
		writeError(c, handler.logger, err)
		return
	}
	signal, _ := json.Marshal(map[string]float64{
		"duration":    job.Clip.EndTime - job.Clip.StartTime,
		"viral_score": viralScore,
	})
	pipe := handler.redis.Pipeline()
	pipe.LPush(c.Request.Context(), "clipstudio:ranking:outcomes", signal)
	pipe.LTrim(c.Request.Context(), "clipstudio:ranking:outcomes", 0, 199)
	_, _ = pipe.Exec(c.Request.Context())
	c.JSON(http.StatusCreated, feedback)
}

func (handler *RenderHandler) ListFeedback(c *gin.Context) {
	userID, _ := middleware.UserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid render job id"})
		return
	}
	if _, err = handler.getOwned(c, userID, id); err != nil {
		writeError(c, handler.logger, err)
		return
	}
	var feedback []model.ClipPerformance
	if err = handler.db.WithContext(c.Request.Context()).
		Where("render_job_id = ?", id).Order("created_at DESC").Find(&feedback).Error; err != nil {
		writeError(c, handler.logger, err)
		return
	}
	c.JSON(http.StatusOK, feedback)
}

func (handler *RenderHandler) findOrCreateVideo(
	c *gin.Context,
	userID uuid.UUID,
	review *service.WorkerAnalysisJob,
) (*model.Video, error) {
	var video model.Video
	err := handler.db.WithContext(c.Request.Context()).Where(
		"user_id = ? AND platform = ? AND external_id = ?",
		userID, review.Platform, review.ExternalID,
	).First(&video).Error
	if err == nil {
		return &video, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	video = model.Video{
		UserID: userID, ExternalID: review.ExternalID, Platform: review.Platform,
		URL: review.URL, Title: review.Title, Thumbnail: review.Thumbnail,
		YouTubeUsername: review.YouTubeUsername,
	}
	video.EnsureID()
	return &video, handler.db.WithContext(c.Request.Context()).Create(&video).Error
}

func (handler *RenderHandler) listOwned(c *gin.Context, userID uuid.UUID) ([]model.RenderJob, error) {
	var jobs []model.RenderJob
	err := handler.db.WithContext(c.Request.Context()).
		Joins("JOIN clips ON clips.id = render_jobs.clip_id").
		Joins("JOIN videos ON videos.id = clips.video_id").
		Where("videos.user_id = ?", userID).
		Preload("Clip").Preload("Clip.Video").
		Order("render_jobs.created_at DESC").Find(&jobs).Error
	return jobs, err
}

func (handler *RenderHandler) getOwned(
	c *gin.Context,
	userID uuid.UUID,
	id uuid.UUID,
) (*model.RenderJob, error) {
	var job model.RenderJob
	err := handler.db.WithContext(c.Request.Context()).
		Joins("JOIN clips ON clips.id = render_jobs.clip_id").
		Joins("JOIN videos ON videos.id = clips.video_id").
		Where("render_jobs.id = ? AND videos.user_id = ?", id, userID).
		Preload("Clip").Preload("Clip.Video").First(&job).Error
	return &job, err
}

func (handler *RenderHandler) sync(c *gin.Context, job *model.RenderJob) {
	if job.Status == model.JobStatusCompleted || job.Status == model.JobStatusFailed {
		return
	}
	state, err := handler.worker.GetRender(c.Request.Context(), job.ID.String())
	if err != nil {
		return
	}
	before, _ := json.Marshal(job)
	applyRenderState(job, state)
	after, _ := json.Marshal(job)
	if bytes.Equal(before, after) {
		return
	}
	_ = handler.db.WithContext(c.Request.Context()).Save(job).Error
	if job.Status == model.JobStatusCompleted {
		_ = handler.db.WithContext(c.Request.Context()).Model(&model.Clip{}).
			Where("id = ?", job.ClipID).Update("status", model.ClipStatusRendered).Error
	}
}

func applyRenderState(job *model.RenderJob, state *service.WorkerRenderJob) {
	job.Status = model.JobStatus(state.Status)
	job.Progress, job.Message = state.Progress, state.Message
	job.OutputPath, job.MediaURL, job.SubtitlePath = state.OutputPath, state.MediaURL, state.SubtitlePath
	if state.Error != nil {
		job.Error = *state.Error
	}
	if state.Marketing != nil {
		job.Title, job.Description = state.Marketing.Title, state.Marketing.Description
		value, _ := json.Marshal(state.Marketing.Hashtags)
		job.Hashtags = string(value)
	}
	if state.Optimization != nil {
		job.Hook = state.Optimization.Hook
		job.RemovedSeconds = state.Optimization.RemovedSeconds
		job.PatternInterrupts = len(state.Optimization.PatternInterrupts)
		job.PlaybackSpeed = state.Optimization.PlaybackSpeed
	}
	if job.Status == model.JobStatusCompleted {
		job.Clip.Status = model.ClipStatusRendered
	}
}

func renderResponse(job model.RenderJob) renderJobResponse {
	var hashtags []string
	_ = json.Unmarshal([]byte(job.Hashtags), &hashtags)
	return renderJobResponse{
		ID: job.ID, ClipID: job.ClipID, ExternalID: job.Clip.Video.ExternalID,
		VideoTitle: job.Clip.Video.Title, Thumbnail: job.Clip.Video.Thumbnail,
		Status: job.Status, Progress: job.Progress, Message: job.Message,
		Score: job.Clip.Score, Duration: job.Clip.EndTime - job.Clip.StartTime,
		OutputPath: job.OutputPath, MediaURL: job.MediaURL,
		Title: job.Title, Description: job.Description, Hashtags: hashtags,
		Error: job.Error, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
		Hook: job.Hook, RemovedSeconds: job.RemovedSeconds,
		PatternInterrupts: job.PatternInterrupts,
		SourceUsername:    job.SourceUsername,
		PlaybackSpeed:     job.PlaybackSpeed,
	}
}
