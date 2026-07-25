package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/middleware"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/repository"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/service"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/sse"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(cfg Config, deps *Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	videoService := service.NewService(
		deps.Repositories.Videos,
		deps.Repositories.AnalysisJobs,
		deps.AsynqClient,
		cfg.AsynqQueue,
	)
	discoveryHTTPClient := &http.Client{Timeout: cfg.DiscoveryTimeout}
	var youtubeProvider service.DiscoveryProvider
	if cfg.EnableYouTubeAPI {
		youtubeProvider = service.NewYouTubeProvider(
			cfg.YouTubeAPIKey,
			discoveryHTTPClient,
			service.YouTubeDiscoveryConfig{
				Region:          cfg.YouTubeRegion,
				Language:        cfg.YouTubeLanguage,
				DefaultQuery:    cfg.YouTubeDiscoveryQuery,
				ReusableOnly:    cfg.YouTubeReusableOnly,
				ExcludeMusic:    cfg.YouTubeExcludeMusic,
				ExcludedTerms:   splitCSV(cfg.YouTubeExcludedTerms),
				DiscoveryWindow: time.Duration(cfg.YouTubeDiscoveryDays) * 24 * time.Hour,
				MinimumDuration: time.Duration(cfg.YouTubeMinDuration) * time.Second,
				MinimumResults:  cfg.DiscoveryMinResults,
			},
		)
	}
	var redditProvider service.DiscoveryProvider
	if cfg.EnableRedditAPI {
		redditProvider = service.NewRedditProvider(
			cfg.RedditClientID,
			cfg.RedditSecret,
			cfg.RedditUserAgent,
			discoveryHTTPClient,
		)
	}
	discoveryService := service.NewDiscoveryService(
		cfg.DiscoveryLimit,
		youtubeProvider,
		redditProvider,
	).WithCache(
		repository.NewRedisDiscoveryCache(deps.Redis),
		cfg.DiscoveryCacheTTL,
	)
	handler := NewHandler(videoService, discoveryService, deps.Logger)
	authHandler := NewAuthHandler(deps.Auth, deps.Logger)
	eventHandler := sse.NewHandler(deps.Repositories.AnalysisJobs, deps.Logger)

	router := gin.New()
	router.Use(
		middleware.RequestIDMiddleware(),
		middleware.LoggerMiddleware(deps.Logger, cfg.Environment != "production"),
		middleware.RecoveryMiddleware(deps.Logger),
		cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{
				"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
			},
			AllowHeaders: []string{
				"Origin", "Content-Type", "Content-Length", "Authorization",
				"X-Request-ID",
			},
			ExposeHeaders: []string{"X-Request-ID"},
		}),
	)

	router.GET("/health", handler.Health)

	auth := router.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)

	videos := router.Group("/videos")
	videos.Use(middleware.Authenticate(deps.Auth))
	videos.GET("/search", handler.SearchVideos)

	api := router.Group("/api")
	api.Use(middleware.Authenticate(deps.Auth))
	api.GET("/jobs/:id/events", eventHandler.JobEvents)
	api.GET("/logs", handler.JobLogs)

	return router
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if term := strings.TrimSpace(part); term != "" {
			result = append(result, term)
		}
	}
	return result
}
