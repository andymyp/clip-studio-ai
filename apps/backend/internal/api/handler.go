package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type Handler struct {
	discovery       *service.DiscoveryService
	recommendations *service.RecommendationService
	validate        *validator.Validate
	logger          *zap.Logger
}

type searchQuery struct {
	Keywords     string `validate:"omitempty,min=2,max=100"`
	Language     string `validate:"omitempty,alpha,min=2,max=10"`
	URL          string `validate:"omitempty,url,max=2048"`
	ContentStyle string `validate:"omitempty,oneof=auto talking_head gameplay comedy emotional livestream cinematic"`
}

func NewHandler(
	discovery *service.DiscoveryService,
	logger *zap.Logger,
	recommendations ...*service.RecommendationService,
) *Handler {
	handler := &Handler{
		discovery: discovery, validate: validator.New(), logger: logger,
	}
	if len(recommendations) > 0 {
		handler.recommendations = recommendations[0]
	}
	return handler
}

func (handler *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (handler *Handler) SearchVideos(c *gin.Context) {
	query := searchQuery{
		Keywords:     strings.TrimSpace(c.Query("keywords")),
		Language:     strings.TrimSpace(c.Query("language")),
		URL:          strings.TrimSpace(c.Query("url")),
		ContentStyle: strings.TrimSpace(c.Query("content_style")),
	}
	if query.Keywords == "" {
		query.Keywords = strings.TrimSpace(c.Query("keyword"))
	}
	if err := handler.validate.Struct(query); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "keywords must contain 2 to 100 characters, language must be valid, and url must be valid",
		})
		return
	}
	if query.Keywords != "" && query.URL != "" {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "provide either keyword or url, not both",
		})
		return
	}
	var results []model.VideoSearchResult
	var err error
	if query.URL == "" && handler.recommendations != nil {
		category := strings.Split(query.Keywords, ",")[0]
		results, err = handler.recommendations.Search(
			c.Request.Context(), query.Language, category, query.ContentStyle,
		)
	} else {
		results, err = handler.discovery.Resolve(c.Request.Context(), query.URL)
	}
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedVideoURL) {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "unsupported video URL"})
			return
		}
		if errors.Is(err, service.ErrNonReusableVideo) {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: "only Creative Commons YouTube videos are supported",
			})
			return
		}
		if errors.Is(err, service.ErrDiscoveryUnavailable) {
			c.JSON(http.StatusServiceUnavailable, errorResponse{
				Error: "video discovery providers are not configured",
			})
			return
		}
		if errors.Is(err, service.ErrDiscoveryRateLimited) {
			c.JSON(http.StatusTooManyRequests, errorResponse{
				Error: "YouTube is temporarily rate limited; try again shortly",
			})
			return
		}
		writeError(c, handler.logger, err)
		return
	}
	c.JSON(http.StatusOK, results)
}

func (handler *Handler) RecommendationStatus(c *gin.Context) {
	if handler.recommendations == nil {
		c.JSON(http.StatusServiceUnavailable, errorResponse{
			Error: "recommendation scheduler is not configured",
		})
		return
	}
	status, err := handler.recommendations.Status(c.Request.Context())
	if err != nil {
		writeError(c, handler.logger, err)
		return
	}
	c.JSON(http.StatusOK, status)
}
