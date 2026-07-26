package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
)

const (
	redditAPIBase  = "https://oauth.reddit.com"
	redditTokenURL = "https://www.reddit.com/api/v1/access_token"
)

type RedditProvider struct {
	clientID     string
	clientSecret string
	userAgent    string
	client       *http.Client
	tokenMu      sync.Mutex
	token        string
	tokenExpires time.Time
}

func NewRedditProvider(
	clientID string,
	clientSecret string,
	userAgent string,
	client *http.Client,
) *RedditProvider {
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
		return nil
	}
	return &RedditProvider{
		clientID: clientID, clientSecret: clientSecret,
		userAgent: userAgent, client: client,
	}
}

func (provider *RedditProvider) Platform() string { return "reddit" }

func (provider *RedditProvider) Supports(link *url.URL) bool {
	host := strings.ToLower(strings.TrimPrefix(link.Hostname(), "www."))
	return host == "reddit.com" || host == "old.reddit.com" || host == "redd.it"
}

func (provider *RedditProvider) Search(
	ctx context.Context,
	keyword string,
	_ string,
	limit int,
) ([]model.VideoSearchResult, error) {
	query := url.Values{
		"q":           {keyword},
		"sort":        {"top"},
		"t":           {"week"},
		"type":        {"link"},
		"limit":       {strconv.Itoa(limit)},
		"raw_json":    {"1"},
		"restrict_sr": {"false"},
	}
	var listing redditListing
	if err := provider.get(ctx, "/search", query, &listing); err != nil {
		return nil, err
	}
	return normalizeRedditPosts(listing.Data.Children), nil
}

func (provider *RedditProvider) Trending(
	ctx context.Context,
	limit int,
) ([]model.VideoSearchResult, error) {
	query := url.Values{
		"limit":    {strconv.Itoa(limit)},
		"raw_json": {"1"},
	}
	var listing redditListing
	if err := provider.get(ctx, "/r/popular/hot", query, &listing); err != nil {
		return nil, err
	}
	return normalizeRedditPosts(listing.Data.Children), nil
}

func (provider *RedditProvider) Resolve(
	ctx context.Context,
	link *url.URL,
) (*model.VideoSearchResult, error) {
	id := redditPostID(link)
	if id == "" {
		return nil, ErrUnsupportedVideoURL
	}
	query := url.Values{"id": {"t3_" + id}, "raw_json": {"1"}}
	var listing redditListing
	if err := provider.get(ctx, "/api/info", query, &listing); err != nil {
		return nil, err
	}
	results := normalizeRedditPosts(listing.Data.Children)
	if len(results) == 0 {
		return nil, fmt.Errorf("post not found")
	}
	return &results[0], nil
}

func (provider *RedditProvider) get(
	ctx context.Context,
	path string,
	query url.Values,
	target any,
) error {
	token, err := provider.accessToken(ctx)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(
		ctx, http.MethodGet, redditAPIBase+path+"?"+query.Encode(), nil,
	)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", provider.userAgent)
	response, err := provider.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("reddit API returned %s", response.Status)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func (provider *RedditProvider) accessToken(ctx context.Context) (string, error) {
	provider.tokenMu.Lock()
	defer provider.tokenMu.Unlock()
	if provider.token != "" && time.Now().Before(provider.tokenExpires) {
		return provider.token, nil
	}

	form := url.Values{"grant_type": {"client_credentials"}}
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, redditTokenURL, strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", err
	}
	request.SetBasicAuth(provider.clientID, provider.clientSecret)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", provider.userAgent)
	response, err := provider.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return "", fmt.Errorf("reddit OAuth returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var token struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil {
		return "", err
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("reddit OAuth returned an empty token")
	}
	provider.token = token.AccessToken
	provider.tokenExpires = time.Now().Add(time.Duration(token.ExpiresIn)*time.Second - time.Minute)
	return provider.token, nil
}

type redditListing struct {
	Data struct {
		Children []struct {
			Data redditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditPost struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	SelfText  string `json:"selftext"`
	URL       string `json:"url"`
	Permalink string `json:"permalink"`
	Thumbnail string `json:"thumbnail"`
	Score     int64  `json:"score"`
	Comments  int64  `json:"num_comments"`
	IsVideo   bool   `json:"is_video"`
	Preview   struct {
		Images []struct {
			Source struct {
				URL string `json:"url"`
			} `json:"source"`
		} `json:"images"`
	} `json:"preview"`
	SecureMedia *struct {
		RedditVideo *struct {
			FallbackURL string  `json:"fallback_url"`
			Duration    float64 `json:"duration"`
		} `json:"reddit_video"`
	} `json:"secure_media"`
}

func normalizeRedditPosts(
	children []struct {
		Data redditPost `json:"data"`
	},
) []model.VideoSearchResult {
	results := make([]model.VideoSearchResult, 0, len(children))
	for _, child := range children {
		post := child.Data
		if !post.IsVideo || post.SecureMedia == nil || post.SecureMedia.RedditVideo == nil {
			continue
		}
		permalink := "https://www.reddit.com" + post.Permalink
		thumbnail := post.Thumbnail
		if len(post.Preview.Images) > 0 {
			thumbnail = post.Preview.Images[0].Source.URL
		}
		var mediaURL string
		var duration float64
		if post.SecureMedia != nil && post.SecureMedia.RedditVideo != nil {
			mediaURL = post.SecureMedia.RedditVideo.FallbackURL
			duration = post.SecureMedia.RedditVideo.Duration
		}
		results = append(results, model.VideoSearchResult{
			ExternalID: post.ID, Platform: "reddit", Title: post.Title,
			Description: post.SelfText, URL: permalink, MediaURL: mediaURL,
			Thumbnail: thumbnail, Duration: duration, Comments: post.Comments,
			Score: post.Score,
		})
	}
	return results
}

func redditPostID(link *url.URL) string {
	host := strings.ToLower(strings.TrimPrefix(link.Hostname(), "www."))
	parts := strings.Split(strings.Trim(link.Path, "/"), "/")
	if host == "redd.it" && len(parts) > 0 {
		return parts[0]
	}
	for index, part := range parts {
		if part == "comments" && index+1 < len(parts) {
			return parts[index+1]
		}
	}
	return ""
}
