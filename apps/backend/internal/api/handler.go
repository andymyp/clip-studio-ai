package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type Handler struct {
	discovery *service.DiscoveryService
	validate  *validator.Validate
	logger    *zap.Logger
}

type searchQuery struct {
	Keywords string `validate:"omitempty,min=2,max=100"`
	Language string `validate:"omitempty,alpha,min=2,max=10"`
	URL      string `validate:"omitempty,url,max=2048"`
}

func NewHandler(
	discovery *service.DiscoveryService,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		discovery: discovery, validate: validator.New(), logger: logger,
	}
}

func (handler *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (handler *Handler) SearchVideos(c *gin.Context) {
	query := searchQuery{
		Keywords: strings.TrimSpace(c.Query("keywords")),
		Language: strings.TrimSpace(c.Query("language")),
		URL:      strings.TrimSpace(c.Query("url")),
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
	results, err := handler.discovery.Discover(
		c.Request.Context(), query.Keywords, query.Language, query.URL,
	)
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedVideoURL) {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "unsupported video URL"})
			return
		}
		if errors.Is(err, service.ErrDiscoveryUnavailable) {
			c.JSON(http.StatusServiceUnavailable, errorResponse{
				Error: "video discovery providers are not configured",
			})
			return
		}
		writeError(c, handler.logger, err)
		return
	}
	c.JSON(http.StatusOK, results)
}
