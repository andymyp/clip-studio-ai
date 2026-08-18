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
	UserID          string `json:"user_id"`
	ExternalID      string `json:"external_id"`
	URL             string `json:"url"`
	Title           string `json:"title"`
	Platform        string `json:"platform"`
	Thumbnail       string `json:"thumbnail"`
	YouTubeUsername string `json:"youtube_username"`
	License         string `json:"license"`
	Reusable        bool   `json:"reusable"`
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
	ID              string       `json:"id"`
	UserID          string       `json:"user_id"`
	ExternalID      string       `json:"external_id"`
	URL             string       `json:"url"`
	Title           string       `json:"title"`
	Platform        string       `json:"platform"`
	Thumbnail       string       `json:"thumbnail"`
	YouTubeUsername string       `json:"youtube_username"`
	License         string       `json:"license"`
	Reusable        bool         `json:"reusable"`
	Status          string       `json:"status"`
	Progress        float64      `json:"progress"`
	Message         string       `json:"message"`
	Clips           []WorkerClip `json:"clips"`
	Error           *string      `json:"error"`
}

type WorkerRenderRequest struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	AnalysisJobID   string  `json:"analysis_job_id"`
	ClipID          string  `json:"clip_id"`
	ClipPath        string  `json:"clip_path"`
	Start           float64 `json:"start"`
	End             float64 `json:"end"`
	WatermarkText   string  `json:"watermark_text"`
	SourceURL       string  `json:"source_url"`
	SourceUsername  string  `json:"source_username"`
	SourceTitle     string  `json:"source_title"`
	PlatformProfile string  `json:"platform_profile"`
	ContentStyle    string  `json:"content_style"`
	RightsConfirmed bool    `json:"rights_confirmed"`
}

type WorkerMarketing struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Hashtags    []string `json:"hashtags"`
}

type WorkerRenderJob struct {
	WorkerRenderRequest
	Status       string              `json:"status"`
	Progress     float64             `json:"progress"`
	Message      string              `json:"message"`
	OutputPath   string              `json:"output_path"`
	MediaURL     string              `json:"media_url"`
	SubtitlePath string              `json:"subtitle_path"`
	Marketing    *WorkerMarketing    `json:"marketing"`
	Optimization *WorkerOptimization `json:"optimization"`
	Quality      *WorkerQuality      `json:"quality"`
	Error        *string             `json:"error"`
}

type WorkerOptimization struct {
	RemovedSeconds    float64   `json:"removed_seconds"`
	Hook              string    `json:"hook"`
	PatternInterrupts []float64 `json:"pattern_interrupts"`
	PlaybackSpeed     float64   `json:"playback_speed"`
	AudioLoudnessLUFS float64   `json:"audio_loudness_lufs"`
}

type WorkerQuality struct {
	Passed          bool    `json:"passed"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	Duration        float64 `json:"duration"`
	AudioVideoDrift float64 `json:"audio_video_drift"`
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

func (client *WorkerClient) CreateRender(
	ctx context.Context,
	payload WorkerRenderRequest,
) (*WorkerRenderJob, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, client.baseURL+"/render-jobs", bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return client.doRender(req)
}

func (client *WorkerClient) GetRender(
	ctx context.Context,
	id string,
) (*WorkerRenderJob, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, client.baseURL+"/render-jobs/"+url.PathEscape(id), nil,
	)
	if err != nil {
		return nil, err
	}
	return client.doRender(req)
}

func (client *WorkerClient) RetryRender(
	ctx context.Context,
	id string,
) (*WorkerRenderJob, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, client.baseURL+"/render-jobs/"+url.PathEscape(id)+"/retry", nil,
	)
	if err != nil {
		return nil, err
	}
	return client.doRender(req)
}

func (client *WorkerClient) doRender(request *http.Request) (*WorkerRenderJob, error) {
	response, err := client.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("worker request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("worker returned %s", response.Status)
	}
	var job WorkerRenderJob
	if err := json.NewDecoder(response.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("decode worker render response: %w", err)
	}
	return &job, nil
}
