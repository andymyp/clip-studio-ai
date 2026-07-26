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
	platform string
	results  []model.VideoSearchResult
	err      error
	supports bool
	calls    int
}

func (provider *fakeDiscoveryProvider) Platform() string { return provider.platform }
func (provider *fakeDiscoveryProvider) Supports(*url.URL) bool {
	return provider.supports
}
func (provider *fakeDiscoveryProvider) Search(
	context.Context,
	string,
	string,
	int,
) ([]model.VideoSearchResult, error) {
	provider.calls++
	return provider.results, provider.err
}
func (provider *fakeDiscoveryProvider) Trending(
	context.Context,
	int,
) ([]model.VideoSearchResult, error) {
	provider.calls++
	return provider.results, provider.err
}

type memoryDiscoveryCache struct {
	values map[string][]model.VideoSearchResult
}

func (cache *memoryDiscoveryCache) Get(
	_ context.Context,
	key string,
) ([]model.VideoSearchResult, bool) {
	results, ok := cache.values[key]
	return results, ok
}

func (cache *memoryDiscoveryCache) Set(
	_ context.Context,
	key string,
	results []model.VideoSearchResult,
	_ time.Duration,
) {
	cache.values[key] = results
}
func (provider *fakeDiscoveryProvider) Resolve(
	context.Context,
	*url.URL,
) (*model.VideoSearchResult, error) {
	if provider.err != nil {
		return nil, provider.err
	}
	return &provider.results[0], nil
}

func TestDiscoveryReturnsPartialProviderResults(t *testing.T) {
	service := NewDiscoveryService(
		10,
		&fakeDiscoveryProvider{
			platform: "youtube",
			results: []model.VideoSearchResult{{
				ExternalID: "video-id",
				Platform:   "youtube",
			}},
		},
		&fakeDiscoveryProvider{platform: "reddit", err: errors.New("unavailable")},
	)

	results, err := service.Discover(context.Background(), "", "", "")

	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(results) != 1 || results[0].ExternalID != "video-id" {
		t.Fatalf("Discover() results = %#v", results)
	}
}

func TestDiscoveryRejectsUnsupportedURL(t *testing.T) {
	service := NewDiscoveryService(
		10,
		&fakeDiscoveryProvider{platform: "youtube"},
	)

	_, err := service.Discover(context.Background(), "", "", "https://example.com/video")

	if !errors.Is(err, ErrUnsupportedVideoURL) {
		t.Fatalf("Discover() error = %v, want ErrUnsupportedVideoURL", err)
	}
}

func TestDiscoveryCachesProviderResults(t *testing.T) {
	provider := &fakeDiscoveryProvider{
		platform: "youtube",
		results: []model.VideoSearchResult{{
			ExternalID: "cached-video",
			Platform:   "youtube",
		}},
	}
	cache := &memoryDiscoveryCache{
		values: make(map[string][]model.VideoSearchResult),
	}
	service := NewDiscoveryService(10, provider).WithCache(cache, time.Hour)

	for range 2 {
		results, err := service.Discover(context.Background(), "", "", "")
		if err != nil || len(results) != 1 {
			t.Fatalf("Discover() results = %#v, error = %v", results, err)
		}
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
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

func TestYouTubeDiscoveryQueryUsesBuiltInFallback(t *testing.T) {
	provider := &YouTubeProvider{
		config: YouTubeDiscoveryConfig{
			ExcludeMusic: true,
		},
	}

	query := provider.discoveryQuery("")

	want := "podcast|interview|education|business|technology|science|story|debate|speech|documentary -music -song -lyrics -album"
	if query != want {
		t.Fatalf("discoveryQuery() = %q, want %q", query, want)
	}
}

func TestYouTubeDiscoveryQueryNormalizesCommaSeparatedKeywords(t *testing.T) {
	provider := &YouTubeProvider{}

	query := provider.discoveryQuery("podcast, interview, education")

	if query != "podcast|interview|education" {
		t.Fatalf("discoveryQuery() = %q", query)
	}
}

func TestContainsExcludedTermMatchesWholeWords(t *testing.T) {
	terms := []string{"religion", "war", "adult content"}
	for _, value := range []string{
		"A discussion about religion and society",
		"Lessons from the war",
		"An adult-content policy overview",
	} {
		if !containsExcludedTerm(value, terms) {
			t.Fatalf("containsExcludedTerm(%q) = false", value)
		}
	}
	if containsExcludedTerm("A software award ceremony", terms) {
		t.Fatal("containsExcludedTerm() matched a partial word")
	}
}
