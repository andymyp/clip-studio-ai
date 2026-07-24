package server

import (
	"net/http"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/config"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/platform"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(cfg config.Config, deps *platform.Dependencies) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), cors.Default())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	api.GET("/events", ssePlaceholder)

	_ = deps
	return router
}

func ssePlaceholder(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.SSEvent("ready", gin.H{"status": "connected"})
}
