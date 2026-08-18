package service

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
)

type fakeDiscoveryProvider struct {
	result   model.VideoSearchResult
	supports bool
}

func (provider *fakeDiscoveryProvider) Platform() string { return "youtube" }
func (provider *fakeDiscoveryProvider) Supports(*url.URL) bool {
	return provider.supports
}
func (provider *fakeDiscoveryProvider) Resolve(
	context.Context,
	*url.URL,
) (*model.VideoSearchResult, error) {
	return &provider.result, nil
}

func TestDiscoveryResolvesSupportedURL(t *testing.T) {
	service := NewDiscoveryService(&fakeDiscoveryProvider{
		supports: true,
		result: model.VideoSearchResult{
			ExternalID: "video-id",
			Platform:   "youtube",
		},
	})

	results, err := service.Resolve(
		context.Background(),
		"https://youtube.com/watch?v=video-id",
	)
	if err != nil || len(results) != 1 || results[0].ExternalID != "video-id" {
		t.Fatalf("Resolve() results = %#v, error = %v", results, err)
	}
}

func TestDiscoveryRejectsUnsupportedURL(t *testing.T) {
	service := NewDiscoveryService(&fakeDiscoveryProvider{})

	_, err := service.Resolve(context.Background(), "https://example.com/video")
	if !errors.Is(err, ErrUnsupportedVideoURL) {
		t.Fatalf("Resolve() error = %v, want ErrUnsupportedVideoURL", err)
	}
}

func TestParseYouTubeDuration(t *testing.T) {
	duration := parseYouTubeDuration("PT1H15M33S")
	if duration.Seconds() != 4533 {
		t.Fatalf("duration = %v", duration)
	}
}

func TestYouTubeVideoID(t *testing.T) {
	for name, rawURL := range map[string]string{
		"watch":  "https://www.youtube.com/watch?v=abc123",
		"short":  "https://youtu.be/abc123",
		"shorts": "https://youtube.com/shorts/abc123",
	} {
		t.Run(name, func(t *testing.T) {
			link, err := url.Parse(rawURL)
			if err != nil {
				t.Fatal(err)
			}
			if id := youtubeVideoID(link); id != "abc123" {
				t.Fatalf("youtubeVideoID() = %q", id)
			}
		})
	}
}

func TestYouTubePageSizeUsesDiscoveryLimit(t *testing.T) {
	for _, test := range []struct {
		limit int
		want  int
	}{
		{limit: -1, want: 1},
		{limit: 20, want: 20},
		{limit: 100, want: 50},
	} {
		if got := youtubePageSize(test.limit); got != test.want {
			t.Fatalf("youtubePageSize(%d) = %d, want %d", test.limit, got, test.want)
		}
	}
}

func TestYouTubeRetryDelayUsesHeaderAndCapsDelay(t *testing.T) {
	if got := youtubeRetryDelay("2", 0); got != 2*time.Second {
		t.Fatalf("youtubeRetryDelay() = %v, want 2s", got)
	}
	if got := youtubeRetryDelay("60", 0); got != 5*time.Second {
		t.Fatalf("youtubeRetryDelay() = %v, want 5s", got)
	}
	if got := youtubeRetryDelay("", 1); got != 2*time.Second {
		t.Fatalf("youtubeRetryDelay() fallback = %v, want 2s", got)
	}
}

func TestYouTubeRequestDelayStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitForYouTubeRequest(ctx, time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForYouTubeRequest() error = %v, want context.Canceled", err)
	}
}

func TestCreativeCommonsOnlyRejectsStandardLicense(t *testing.T) {
	results := creativeCommonsOnly([]model.VideoSearchResult{
		{ExternalID: "cc", License: "creativeCommon", Reusable: true},
		{ExternalID: "standard", License: "youtube", Reusable: false},
	})
	if len(results) != 1 || results[0].ExternalID != "cc" {
		t.Fatalf("creativeCommonsOnly() = %#v", results)
	}
}
