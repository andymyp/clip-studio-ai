package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
)

var (
	ErrDiscoveryUnavailable = errors.New("video discovery is not configured")
	ErrUnsupportedVideoURL  = errors.New("unsupported video URL")
)

type DiscoveryProvider interface {
	Platform() string
	Search(context.Context, string, int) ([]model.VideoSearchResult, error)
	Trending(context.Context, int) ([]model.VideoSearchResult, error)
	Resolve(context.Context, *url.URL) (*model.VideoSearchResult, error)
	Supports(*url.URL) bool
}

type DiscoveryService struct {
	providers []DiscoveryProvider
	limit     int
	cache     DiscoveryCache
	cacheTTL  time.Duration
}

type DiscoveryCache interface {
	Get(context.Context, string) ([]model.VideoSearchResult, bool)
	Set(context.Context, string, []model.VideoSearchResult, time.Duration)
}

func NewDiscoveryService(limit int, providers ...DiscoveryProvider) *DiscoveryService {
	enabled := make([]DiscoveryProvider, 0, len(providers))
	for _, provider := range providers {
		if provider != nil {
			enabled = append(enabled, provider)
		}
	}
	return &DiscoveryService{providers: enabled, limit: limit}
}

func (service *DiscoveryService) WithCache(
	cache DiscoveryCache,
	ttl time.Duration,
) *DiscoveryService {
	service.cache = cache
	service.cacheTTL = ttl
	return service
}

func (service *DiscoveryService) Discover(
	ctx context.Context,
	keyword string,
	rawURL string,
) ([]model.VideoSearchResult, error) {
	if len(service.providers) == 0 {
		return nil, ErrDiscoveryUnavailable
	}
	cacheKey := discoveryCacheKey(keyword, rawURL)
	if service.cache != nil && service.cacheTTL > 0 {
		if cached, ok := service.cache.Get(ctx, cacheKey); ok {
			return cached, nil
		}
	}

	var results []model.VideoSearchResult
	var err error
	if rawURL != "" {
		results, err = service.resolve(ctx, rawURL)
	} else {
		results, err = service.discoverProviders(ctx, keyword)
	}
	if err != nil {
		return nil, err
	}
	if service.cache != nil && service.cacheTTL > 0 {
		service.cache.Set(ctx, cacheKey, results, service.cacheTTL)
	}
	return results, nil
}

func (service *DiscoveryService) discoverProviders(
	ctx context.Context,
	keyword string,
) ([]model.VideoSearchResult, error) {
	type response struct {
		results []model.VideoSearchResult
		err     error
	}
	responses := make(chan response, len(service.providers))
	var wait sync.WaitGroup
	for _, provider := range service.providers {
		wait.Add(1)
		go func(provider DiscoveryProvider) {
			defer wait.Done()
			var results []model.VideoSearchResult
			var err error
			if keyword == "" {
				results, err = provider.Trending(ctx, service.limit)
			} else {
				results, err = provider.Search(ctx, keyword, service.limit)
			}
			responses <- response{results: results, err: err}
		}(provider)
	}
	wait.Wait()
	close(responses)

	results := make([]model.VideoSearchResult, 0, service.limit*len(service.providers))
	var failures []error
	for response := range responses {
		if response.err != nil {
			failures = append(failures, response.err)
			continue
		}
		results = append(results, response.results...)
	}
	if len(results) == 0 && len(failures) > 0 {
		return nil, fmt.Errorf("discover videos: %w", errors.Join(failures...))
	}
	if len(results) > service.limit {
		results = results[:service.limit]
	}
	return results, nil
}

func discoveryCacheKey(keyword, rawURL string) string {
	value := "v3\x00" + strings.ToLower(strings.TrimSpace(keyword)) + "\x00" + strings.TrimSpace(rawURL)
	hash := sha256.Sum256([]byte(value))
	return fmt.Sprintf("discovery:%x", hash)
}

func (service *DiscoveryService) resolve(
	ctx context.Context,
	rawURL string,
) ([]model.VideoSearchResult, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, ErrUnsupportedVideoURL
	}
	for _, provider := range service.providers {
		if !provider.Supports(parsed) {
			continue
		}
		result, err := provider.Resolve(ctx, parsed)
		if err != nil {
			return nil, fmt.Errorf("resolve %s video: %w", provider.Platform(), err)
		}
		return []model.VideoSearchResult{*result}, nil
	}
	return nil, ErrUnsupportedVideoURL
}
