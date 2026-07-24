package api

import (
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/middleware"
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
	handler := NewHandler(videoService, deps.Logger)
	eventHandler := sse.NewHandler(deps.Repositories.AnalysisJobs, deps.Logger)

	router := gin.New()
	router.Use(
		middleware.RequestIDMiddleware(),
		middleware.LoggerMiddleware(deps.Logger, cfg.Environment != "production"),
		middleware.RecoveryMiddleware(deps.Logger),
		cors.Default(),
	)

	router.GET("/health", handler.Health)
	api := router.Group("/api")
	api.GET("/videos/search", handler.SearchVideos)
	api.GET("/videos/:id", handler.GetVideo)
	api.POST("/videos/:id/analyze", handler.AnalyzeVideo)
	api.GET("/jobs/:id/events", eventHandler.JobEvents)

	return router
}
