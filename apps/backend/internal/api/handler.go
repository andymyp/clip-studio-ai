package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/middleware"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	videos   *service.Service
	validate *validator.Validate
	logger   *zap.Logger
}

type searchQuery struct {
	Keyword string `validate:"required,min=2,max=100"`
}

type videoSearchResponse struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail"`
	Views     int64  `json:"views"`
	Platform  string `json:"platform"`
}

type videoDetailResponse struct {
	ID          uuid.UUID `json:"id"`
	Platform    string    `json:"platform"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Thumbnail   string    `json:"thumbnail"`
	Duration    float64   `json:"duration"`
	Views       int64     `json:"views"`
	Likes       int64     `json:"likes"`
	Comments    int64     `json:"comments"`
	Transcript  string    `json:"transcript"`
	CreatedAt   time.Time `json:"created_at"`
}

type analysisResponse struct {
	ID       uuid.UUID       `json:"id"`
	VideoID  uuid.UUID       `json:"video_id"`
	Status   model.JobStatus `json:"status"`
	Progress float64         `json:"progress"`
	Message  string          `json:"message"`
}

func NewHandler(
	videos *service.Service,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		videos: videos, validate: validator.New(), logger: logger,
	}
}

func (handler *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (handler *Handler) SearchVideos(c *gin.Context) {
	query := searchQuery{Keyword: strings.TrimSpace(c.Query("keyword"))}
	if err := handler.validate.Struct(query); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "keyword must contain between 2 and 100 characters",
		})
		return
	}

	userID, _ := middleware.UserID(c)
	videos, err := handler.videos.Search(c.Request.Context(), userID, query.Keyword)
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}

	response := make([]videoSearchResponse, 0, len(videos))
	for _, video := range videos {
		response = append(response, videoSearchResponse{
			Title: video.Title, URL: video.URL, Thumbnail: video.Thumbnail,
			Views: video.Views, Platform: video.Platform,
		})
	}
	c.JSON(http.StatusOK, response)
}

func (handler *Handler) GetVideo(c *gin.Context) {
	id, ok := handler.parseUUID(c, "id")
	if !ok {
		return
	}
	userID, _ := middleware.UserID(c)
	video, err := handler.videos.Get(c.Request.Context(), userID, id)
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	c.JSON(http.StatusOK, videoDetailResponse{
		ID: video.ID, Platform: video.Platform, URL: video.URL, Title: video.Title,
		Description: video.Description, Thumbnail: video.Thumbnail, Duration: video.Duration,
		Views: video.Views, Likes: video.Likes, Comments: video.Comments,
		Transcript: video.Transcript, CreatedAt: video.CreatedAt,
	})
}

func (handler *Handler) AnalyzeVideo(c *gin.Context) {
	id, ok := handler.parseUUID(c, "id")
	if !ok {
		return
	}
	userID, _ := middleware.UserID(c)
	job, err := handler.videos.Analyze(c.Request.Context(), userID, id)
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	c.JSON(http.StatusAccepted, analysisResponse{
		ID: job.ID, VideoID: job.VideoID, Status: job.Status,
		Progress: job.Progress, Message: job.Message,
	})
}

func (handler *Handler) parseUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	value := c.Param(name)
	if err := handler.validate.Var(value, "required,uuid4"); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: name + " must be a valid UUID"})
		return uuid.Nil, false
	}
	id, err := uuid.Parse(value)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: name + " must be a valid UUID"})
		return uuid.Nil, false
	}
	return id, true
}
