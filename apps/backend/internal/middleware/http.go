package middleware

import (
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const requestIDKey = "request_id"

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func RequestID(c *gin.Context) string {
	value, _ := c.Get(requestIDKey)
	requestID, _ := value.(string)
	return requestID
}

func LoggerMiddleware(logger *zap.Logger, color bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		status := c.Writer.Status()
		message := "http request"
		if color {
			message += " " + coloredStatus(status)
		}
		fields := []zap.Field{
			zap.String("request_id", RequestID(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", status),
			zap.Duration("latency", time.Since(started)),
		}
		logger.Info(message, fields...)
	}
}

func coloredStatus(status int) string {
	color := "\x1b[31m"
	switch {
	case status < http.StatusMultipleChoices:
		color = "\x1b[32m"
	case status < http.StatusBadRequest:
		color = "\x1b[36m"
	case status < http.StatusInternalServerError:
		color = "\x1b[33m"
	}
	return color + strconv.Itoa(status) + "\x1b[0m"
}

func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered",
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()),
					zap.String("request_id", RequestID(c)),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()
		c.Next()
	}
}
