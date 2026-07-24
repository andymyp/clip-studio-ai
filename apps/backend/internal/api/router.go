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

	api := router.Group("/api")
	api.Use(middleware.Authenticate(deps.Auth))
	api.GET("/videos/search", handler.SearchVideos)
	api.GET("/videos/:id", handler.GetVideo)
	api.POST("/videos/:id/analyze", handler.AnalyzeVideo)
	api.GET("/jobs/:id/events", eventHandler.JobEvents)

	return router
}
