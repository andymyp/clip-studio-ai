package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(nil, zap.NewNop())
	router := gin.New()
	router.GET("/health", handler.Health)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusOK || recorder.Body.String() != `{"status":"ok"}` {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}
