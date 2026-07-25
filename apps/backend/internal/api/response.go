package api

import (
	"errors"
	"net/http"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/middleware"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/repository"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(c *gin.Context, logger *zap.Logger, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, errorResponse{Error: "resource not found"})
		return
	}

	logger.Error("request failed",
		zap.Error(err),
		zap.String("request_id", middleware.RequestID(c)),
	)
	c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal server error"})
}
