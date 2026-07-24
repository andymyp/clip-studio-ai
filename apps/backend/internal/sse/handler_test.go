package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type staticJobRepository struct {
	job *model.AnalysisJob
}

func (repo *staticJobRepository) Create(context.Context, *model.AnalysisJob) error { return nil }
func (repo *staticJobRepository) GetByID(context.Context, uuid.UUID) (*model.AnalysisJob, error) {
	return repo.job, nil
}
func (repo *staticJobRepository) GetByIDForUser(
	context.Context,
	uuid.UUID,
	uuid.UUID,
) (*model.AnalysisJob, error) {
	return repo.job, nil
}
func (repo *staticJobRepository) ListByVideoID(context.Context, uuid.UUID) ([]model.AnalysisJob, error) {
	return nil, nil
}
func (repo *staticJobRepository) Update(context.Context, *model.AnalysisJob) error { return nil }
func (repo *staticJobRepository) Delete(context.Context, uuid.UUID) error          { return nil }

func TestJobEventsSendsTerminalProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jobID := uuid.New()
	handler := NewHandler(&staticJobRepository{job: &model.AnalysisJob{
		Base: model.Base{ID: jobID}, Status: model.JobStatusCompleted,
		Progress: 100, Message: "complete",
	}}, zap.NewNop())
	router := gin.New()
	router.GET("/api/jobs/:id/events", handler.JobEvents)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/jobs/"+jobID.String()+"/events", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "event:completed") ||
		!strings.Contains(body, `"percentage":100`) {
		t.Fatalf("unexpected SSE body: %q", body)
	}
}
