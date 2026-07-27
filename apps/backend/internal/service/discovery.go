package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
)

var (
	ErrDiscoveryUnavailable = errors.New("video resolver is not configured")
	ErrDiscoveryRateLimited = errors.New("video discovery rate limited")
	ErrNonReusableVideo     = errors.New("video is not Creative Commons licensed")
	ErrUnsupportedVideoURL  = errors.New("unsupported video URL")
)

type DiscoveryProvider interface {
	Platform() string
	Resolve(context.Context, *url.URL) (*model.VideoSearchResult, error)
	Supports(*url.URL) bool
}

type DiscoveryService struct {
	providers []DiscoveryProvider
}

func NewDiscoveryService(providers ...DiscoveryProvider) *DiscoveryService {
	enabled := make([]DiscoveryProvider, 0, len(providers))
	for _, provider := range providers {
		if provider != nil {
			enabled = append(enabled, provider)
		}
	}
	return &DiscoveryService{providers: enabled}
}

func (service *DiscoveryService) Resolve(
	ctx context.Context,
	rawURL string,
) ([]model.VideoSearchResult, error) {
	if len(service.providers) == 0 {
		return nil, ErrDiscoveryUnavailable
	}
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
