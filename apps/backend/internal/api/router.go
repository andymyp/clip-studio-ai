package api

import (
	"net/http"
	"strings"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/middleware"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(cfg Config, deps *Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	discoveryHTTPClient := &http.Client{Timeout: cfg.DiscoveryTimeout}
	var youtubeProvider *service.YouTubeProvider
	if cfg.EnableYouTubeAPI {
		youtubeProvider = service.NewYouTubeProvider(
			cfg.YouTubeAPIKey,
			discoveryHTTPClient,
			service.YouTubeDiscoveryConfig{
				Region: cfg.YouTubeRegion,
			},
		)
	}
	discoveryService := service.NewDiscoveryService(
		youtubeProvider,
	)
	recommendationService := service.NewRecommendationService(
		deps.DB, deps.Redis, cfg.DiscoveryCacheTTL, cfg.DiscoveryLimit,
	)
	handler := NewHandler(discoveryService, deps.Logger, recommendationService)
	if youtubeProvider != nil {
		deps.Recommendations = service.NewRecommendationScheduler(
			youtubeProvider,
			recommendationService,
			splitCSV(cfg.RecommendationKeywords),
			cfg.RecommendationInterval,
			50,
			cfg.RecommendationKeywordsPerRun,
			deps.Logger,
		)
		deps.Recommendations.Start()
	}
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
	videos.GET("/recommendations", handler.SearchVideos)
	videos.GET("/recommendations/status", handler.RecommendationStatus)

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
