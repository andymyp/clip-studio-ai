package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type WorkerClient struct {
	baseURL string
	client  *http.Client
}

type WorkerAnalyzeRequest struct {
	UserID     string `json:"user_id"`
	ExternalID string `json:"external_id"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	Platform   string `json:"platform"`
	Thumbnail  string `json:"thumbnail"`
}

type WorkerClip struct {
	ID         string  `json:"id"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Score      float64 `json:"score"`
	Reason     string  `json:"reason"`
	OutputPath string  `json:"output_path"`
	MediaURL   string  `json:"media_url"`
}

type WorkerAnalysisJob struct {
	ID         string       `json:"id"`
	UserID     string       `json:"user_id"`
	ExternalID string       `json:"external_id"`
	URL        string       `json:"url"`
	Title      string       `json:"title"`
	Platform   string       `json:"platform"`
	Thumbnail  string       `json:"thumbnail"`
	Status     string       `json:"status"`
	Progress   float64      `json:"progress"`
	Message    string       `json:"message"`
	Clips      []WorkerClip `json:"clips"`
	Error      *string      `json:"error"`
}

func NewWorkerClient(baseURL string, client *http.Client) *WorkerClient {
	return &WorkerClient{baseURL: strings.TrimRight(baseURL, "/"), client: client}
}

func (client *WorkerClient) CreateAnalysis(
	ctx context.Context,
	payload WorkerAnalyzeRequest,
) (*WorkerAnalysisJob, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, client.baseURL+"/jobs", bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	return client.do(request)
}

func (client *WorkerClient) GetAnalysis(
	ctx context.Context,
	id string,
) (*WorkerAnalysisJob, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		client.baseURL+"/jobs/"+url.PathEscape(id),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return client.do(request)
}

func (client *WorkerClient) GetReview(
	ctx context.Context,
	userID string,
	externalID string,
) (*WorkerAnalysisJob, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		client.baseURL+"/reviews/"+url.PathEscape(userID)+"/"+url.PathEscape(externalID),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return client.do(request)
}

func (client *WorkerClient) do(request *http.Request) (*WorkerAnalysisJob, error) {
	response, err := client.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("worker request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("worker returned %s", response.Status)
	}
	var job WorkerAnalysisJob
	if err := json.NewDecoder(response.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("decode worker response: %w", err)
	}
	return &job, nil
}
