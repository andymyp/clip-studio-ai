package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/middleware"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/repository"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(cfg Config, deps *Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

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
	handler := NewHandler(discoveryService, deps.Logger)
	workerClient := service.NewWorkerClient(cfg.WorkerAPIURL, discoveryHTTPClient)
	workerHandler := NewWorkerHandler(workerClient, deps.Logger)
	renderHandler := NewRenderHandler(deps.DB, deps.Redis, workerClient, deps.Logger)
	authHandler := NewAuthHandler(deps.Auth, deps.Logger)

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

	clips := router.Group("/clips")
	clips.Use(middleware.Authenticate(deps.Auth))
	clips.POST("/analyze", workerHandler.CreateAnalysis)
	clips.GET("/jobs/:id", workerHandler.GetAnalysis)
	clips.GET("/reviews/:external_id", workerHandler.GetReview)
	clips.GET("/reviews/:external_id/events", workerHandler.AnalysisEvents)
	clips.GET("/rendered", renderHandler.Rendered)

	renders := router.Group("/renders")
	renders.Use(middleware.Authenticate(deps.Auth))
	renders.POST("", renderHandler.Create)
	renders.GET("", renderHandler.List)
	renders.GET("/events", renderHandler.Events)
	renders.POST("/:id/retry", renderHandler.Retry)
	renders.POST("/:id/feedback", renderHandler.CreateFeedback)
	renders.GET("/:id/feedback", renderHandler.ListFeedback)

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
