package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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
	RightsConfirmed bool `json:"rights_confirmed" binding:"required"`
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
	PlatformProfile   string          `json:"platform_profile"`
	RightsConfirmed   bool            `json:"rights_confirmed"`
	QualityPassed     bool            `json:"quality_passed"`
	OutputWidth       int             `json:"output_width"`
	OutputHeight      int             `json:"output_height"`
	OutputDuration    float64         `json:"output_duration"`
	AudioVideoDrift   float64         `json:"audio_video_drift"`
	AudioLoudnessLUFS float64         `json:"audio_loudness_lufs"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type renderJobPageResponse struct {
	Items      []renderJobResponse `json:"items"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalItems int64               `json:"total_items"`
	TotalPages int                 `json:"total_pages"`
}

type createFeedbackRequest struct {
	Platform      string  `json:"platform" binding:"required,oneof=youtube tiktok instagram"`
	Views         int64   `json:"views" binding:"min=0"`
	Likes         int64   `json:"likes" binding:"min=0"`
	Comments      int64   `json:"comments" binding:"min=0"`
	Shares        int64   `json:"shares" binding:"min=0"`
	AverageWatch  float64 `json:"average_watch_seconds" binding:"min=0"`
	CompletionPct float64 `json:"completion_percentage" binding:"min=0,max=100"`
	EngagedViews  int64   `json:"engaged_views" binding:"min=0"`
	SwipedAwayPct float64 `json:"swiped_away_percentage" binding:"min=0,max=100"`
	Replays       int64   `json:"replays" binding:"min=0"`
	DropoffSecond float64 `json:"dropoff_second" binding:"min=0"`
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
		c.JSON(http.StatusBadRequest, errorResponse{Error: "clips, platform profile, and rights confirmation are required"})
		return
	}
	if !body.RightsConfirmed {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "confirm that you have permission to reuse this content"})
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
			PlatformProfile: "smart", RightsConfirmed: body.RightsConfirmed,
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
			SourceTitle: review.Title,
			PlatformProfile: job.PlatformProfile, RightsConfirmed: job.RightsConfirmed,
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
	page := queryInt(c, "page", 1, 1, 1_000_000)
	pageSize := queryInt(c, "page_size", 10, 1, 100)
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if completedOnly {
		status = string(model.JobStatusCompleted)
	} else if status != "" && status != "all" && !validRenderStatus(status) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid render status"})
		return
	}
	sort := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort", "newest")))
	order, found := renderSortOrders[sort]
	if !found {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid render sort"})
		return
	}
	query := handler.ownedQuery(c, userID)
	if status != "" && status != "all" {
		query = query.Where("render_jobs.status = ?", status)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(render_jobs.title) LIKE ? OR LOWER(render_jobs.description) LIKE ? OR LOWER(videos.title) LIKE ? OR LOWER(render_jobs.source_username) LIKE ?",
			pattern, pattern, pattern, pattern,
		)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		writeError(c, handler.logger, err)
		return
	}
	var jobs []model.RenderJob
	err := query.Preload("Clip").Preload("Clip.Video").
		Order(order).Offset((page - 1) * pageSize).Limit(pageSize).Find(&jobs).Error
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
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	c.JSON(http.StatusOK, renderJobPageResponse{
		Items: responses, Page: page, PageSize: pageSize,
		TotalItems: total, TotalPages: totalPages,
	})
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
	engagedRate := float64(body.EngagedViews) / float64(max(body.Views, 1)) * 100
	replayRate := float64(body.Replays) / float64(max(body.Views, 1)) * 100
	viralScore := min(100, body.CompletionPct*0.35+(100-body.SwipedAwayPct)*0.20+
		min(engagedRate, 100)*0.15+min(replayRate, 100)*0.10+
		min(engagement, 100)*0.15+
		min(body.AverageWatch/max(job.Clip.EndTime-job.Clip.StartTime, 1)*100, 100)*0.05)
	feedback := model.ClipPerformance{
		RenderJobID: id, Platform: body.Platform, Views: body.Views, Likes: body.Likes,
		Comments: body.Comments, Shares: body.Shares, AverageWatch: body.AverageWatch,
		CompletionPct: body.CompletionPct, ViralScore: viralScore,
		EngagedViews: body.EngagedViews, SwipedAwayPct: body.SwipedAwayPct,
		Replays: body.Replays, DropoffSecond: body.DropoffSecond,
	}
	feedback.EnsureID()
	if err = handler.db.WithContext(c.Request.Context()).Create(&feedback).Error; err != nil {
		writeError(c, handler.logger, err)
		return
	}
	signal, _ := json.Marshal(map[string]float64{
		"duration":    job.Clip.EndTime - job.Clip.StartTime,
		"viral_score": viralScore,
		"engaged_rate": engagedRate,
		"replay_rate": replayRate,
		"swiped_away_percentage": body.SwipedAwayPct,
		"dropoff_second": body.DropoffSecond,
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
		License: review.License, Reusable: review.Reusable,
	}
	video.EnsureID()
	return &video, handler.db.WithContext(c.Request.Context()).Create(&video).Error
}

func (handler *RenderHandler) listOwned(c *gin.Context, userID uuid.UUID) ([]model.RenderJob, error) {
	var jobs []model.RenderJob
	err := handler.ownedQuery(c, userID).
		Preload("Clip").Preload("Clip.Video").
		Order("render_jobs.created_at DESC").Find(&jobs).Error
	return jobs, err
}

func (handler *RenderHandler) ownedQuery(c *gin.Context, userID uuid.UUID) *gorm.DB {
	return handler.db.WithContext(c.Request.Context()).Model(&model.RenderJob{}).
		Joins("JOIN clips ON clips.id = render_jobs.clip_id").
		Joins("JOIN videos ON videos.id = clips.video_id").
		Where("videos.user_id = ?", userID)
}

var renderSortOrders = map[string]string{
	"newest":   "render_jobs.created_at DESC, render_jobs.id DESC",
	"oldest":   "render_jobs.created_at ASC, render_jobs.id ASC",
	"score":    "clips.score DESC, render_jobs.created_at DESC",
	"progress": "render_jobs.progress DESC, render_jobs.created_at DESC",
}

func validRenderStatus(status string) bool {
	switch model.JobStatus(status) {
	case model.JobStatusQueued, model.JobStatusPending, model.JobStatusProcessing,
		model.JobStatusCompleted, model.JobStatusFailed, model.JobStatusCancelled:
		return true
	default:
		return false
	}
}

func queryInt(c *gin.Context, key string, fallback, minimum, maximum int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value < minimum {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
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
		job.AudioLoudnessLUFS = state.Optimization.AudioLoudnessLUFS
	}
	if state.Quality != nil {
		job.QualityPassed = state.Quality.Passed
		job.OutputWidth, job.OutputHeight = state.Quality.Width, state.Quality.Height
		job.OutputDuration, job.AudioVideoDrift = state.Quality.Duration, state.Quality.AudioVideoDrift
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
		PlatformProfile:   job.PlatformProfile,
		RightsConfirmed:   job.RightsConfirmed,
		QualityPassed:     job.QualityPassed,
		OutputWidth:       job.OutputWidth,
		OutputHeight:      job.OutputHeight,
		OutputDuration:    job.OutputDuration,
		AudioVideoDrift:   job.AudioVideoDrift,
		AudioLoudnessLUFS: job.AudioLoudnessLUFS,
	}
}
