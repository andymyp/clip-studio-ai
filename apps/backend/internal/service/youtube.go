package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
)

const youtubeAPIBase = "https://www.googleapis.com/youtube/v3"

var youtubeDurationPattern = regexp.MustCompile(`^P(?:(\d+)D)?T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

type YouTubeProvider struct {
	apiKey string
	client *http.Client
	config YouTubeDiscoveryConfig
}

type YouTubeDiscoveryConfig struct {
	Region          string
	Language        string
	ExcludeMusic    bool
	ExcludedTerms   []string
	DiscoveryWindow time.Duration
	MinimumDuration time.Duration
	MinimumResults  int
}

func NewYouTubeProvider(
	apiKey string,
	client *http.Client,
	config YouTubeDiscoveryConfig,
) *YouTubeProvider {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	return &YouTubeProvider{apiKey: apiKey, client: client, config: config}
}

func (provider *YouTubeProvider) Platform() string { return "youtube" }

func (provider *YouTubeProvider) Supports(link *url.URL) bool {
	host := strings.ToLower(strings.TrimPrefix(link.Hostname(), "www."))
	return host == "youtube.com" || host == "m.youtube.com" || host == "youtu.be"
}

func (provider *YouTubeProvider) Search(
	ctx context.Context,
	keyword string,
	language string,
	limit int,
) ([]model.VideoSearchResult, error) {
	query := url.Values{
		"part":            {"snippet"},
		"type":            {"video"},
		"videoEmbeddable": {"true"},
		"videoSyndicated": {"true"},
		"chart":           {"mostPopular"},
		"order":           {"viewCount"},
		"maxResults":      {"50"},
		"q":               {provider.discoveryQuery(keyword)},
		"safeSearch":      {"strict"},
		"videoLicense":    {"creativeCommon"},
		"key":             {provider.apiKey},
	}
	if provider.config.Region != "" {
		query.Set("regionCode", provider.config.Region)
	}
	relevanceLanguage := strings.TrimSpace(language)
	if relevanceLanguage == "" {
		relevanceLanguage = provider.config.Language
	}
	if relevanceLanguage != "" {
		query.Set("relevanceLanguage", relevanceLanguage)
	}
	if provider.config.DiscoveryWindow > 0 {
		query.Set(
			"publishedAfter",
			time.Now().UTC().Add(-provider.config.DiscoveryWindow).Format(time.RFC3339),
		)
	}
	minimum := min(provider.config.MinimumResults, limit)
	results := make([]model.VideoSearchResult, 0, limit)
	for page := 0; page < 2 && len(results) < max(minimum, 1); page++ {
		var response youtubeSearchResponse
		if err := provider.get(ctx, "/search", query, &response); err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(response.Items))
		for _, item := range response.Items {
			if item.ID.VideoID != "" {
				ids = append(ids, item.ID.VideoID)
			}
		}
		details, err := provider.videos(ctx, ids)
		if err != nil {
			return nil, err
		}
		results = append(
			results,
			provider.filterDiscoveryResults(details, limit-len(results))...,
		)
		if response.NextPageToken == "" {
			break
		}
		query.Set("pageToken", response.NextPageToken)
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (provider *YouTubeProvider) Trending(
	ctx context.Context,
	limit int,
) ([]model.VideoSearchResult, error) {
	return provider.Search(ctx, "", "", limit)
}

func (provider *YouTubeProvider) Resolve(
	ctx context.Context,
	link *url.URL,
) (*model.VideoSearchResult, error) {
	id := youtubeVideoID(link)
	if id == "" {
		return nil, ErrUnsupportedVideoURL
	}
	results, err := provider.videos(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("video not found")
	}
	return &results[0], nil
}

func (provider *YouTubeProvider) videos(
	ctx context.Context,
	ids []string,
) ([]model.VideoSearchResult, error) {
	if len(ids) == 0 {
		return []model.VideoSearchResult{}, nil
	}
	query := url.Values{
		"part": {"snippet,contentDetails,statistics,status"},
		"id":   {strings.Join(ids, ",")},
		"key":  {provider.apiKey},
	}
	var response youtubeVideosResponse
	if err := provider.get(ctx, "/videos", query, &response); err != nil {
		return nil, err
	}
	results := normalizeYouTubeVideos(response.Items)
	channelIDs := make([]string, 0, len(results))
	for _, result := range results {
		if result.ChannelID != "" {
			channelIDs = append(channelIDs, result.ChannelID)
		}
	}
	handles, err := provider.channelHandles(ctx, channelIDs)
	if err == nil {
		for index := range results {
			results[index].YouTubeUsername = handles[results[index].ChannelID]
		}
	}
	return results, nil
}

func (provider *YouTubeProvider) channelHandles(
	ctx context.Context,
	ids []string,
) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	query := url.Values{
		"part": {"snippet"},
		"id":   {strings.Join(ids, ",")},
		"key":  {provider.apiKey},
	}
	var response youtubeChannelsResponse
	if err := provider.get(ctx, "/channels", query, &response); err != nil {
		return nil, err
	}
	handles := make(map[string]string, len(response.Items))
	for _, channel := range response.Items {
		handle := strings.TrimSpace(channel.Snippet.CustomURL)
		if handle != "" && !strings.HasPrefix(handle, "@") {
			handle = "@" + handle
		}
		handles[channel.ID] = handle
	}
	return handles, nil
}

func (provider *YouTubeProvider) get(
	ctx context.Context,
	path string,
	query url.Values,
	target any,
) error {
	request, err := http.NewRequestWithContext(
		ctx, http.MethodGet, youtubeAPIBase+path+"?"+query.Encode(), nil,
	)
	if err != nil {
		return err
	}
	response, err := provider.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("youtube API returned %s", response.Status)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

type youtubeSearchResponse struct {
	NextPageToken string `json:"nextPageToken"`
	Items         []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"items"`
}

type youtubeVideosResponse struct {
	Items []youtubeVideo `json:"items"`
}

type youtubeChannelsResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			CustomURL string `json:"customUrl"`
		} `json:"snippet"`
	} `json:"items"`
}

type youtubeVideo struct {
	ID      string `json:"id"`
	Snippet struct {
		Title        string `json:"title"`
		Description  string `json:"description"`
		CategoryID   string `json:"categoryId"`
		ChannelID    string `json:"channelId"`
		ChannelTitle string `json:"channelTitle"`
		Thumbnails   map[string]struct {
			URL string `json:"url"`
		} `json:"thumbnails"`
	} `json:"snippet"`
	ContentDetails struct {
		Duration string `json:"duration"`
	} `json:"contentDetails"`
	Statistics struct {
		ViewCount    string `json:"viewCount"`
		LikeCount    string `json:"likeCount"`
		CommentCount string `json:"commentCount"`
	} `json:"statistics"`
	Status struct {
		License    string `json:"license"`
		Embeddable bool   `json:"embeddable"`
	} `json:"status"`
}

func normalizeYouTubeVideos(items []youtubeVideo) []model.VideoSearchResult {
	results := make([]model.VideoSearchResult, 0, len(items))
	for _, item := range items {
		results = append(results, model.VideoSearchResult{
			ExternalID: item.ID, Platform: "youtube", Title: item.Snippet.Title,
			CategoryID:   item.Snippet.CategoryID,
			ChannelID:    item.Snippet.ChannelID,
			ChannelTitle: item.Snippet.ChannelTitle,
			Description:  item.Snippet.Description,
			URL:          "https://www.youtube.com/watch?v=" + item.ID,
			EmbedURL:     "https://www.youtube.com/embed/" + item.ID,
			Thumbnail:    bestYouTubeThumbnail(item.Snippet.Thumbnails),
			Duration:     parseYouTubeDuration(item.ContentDetails.Duration).Seconds(),
			Views:        parseMetric(item.Statistics.ViewCount),
			Likes:        parseMetric(item.Statistics.LikeCount),
			Comments:     parseMetric(item.Statistics.CommentCount),
			License:      item.Status.License,
			Reusable:     item.Status.License == "creativeCommon",
		})
	}
	return results
}

func (provider *YouTubeProvider) discoveryQuery(keyword string) string {
	base := normalizeYouTubeKeywords(keyword)
	if provider.config.ExcludeMusic {
		base += " -music -song -lyrics -album"
	}
	return base
}

func normalizeYouTubeKeywords(value string) string {
	parts := strings.FieldsFunc(value, func(character rune) bool {
		return character == ',' || character == '|'
	})
	keywords := make([]string, 0, len(parts))
	for _, part := range parts {
		if keyword := strings.TrimSpace(part); keyword != "" {
			keywords = append(keywords, keyword)
		}
	}
	return strings.Join(keywords, "|")
}

func (provider *YouTubeProvider) filterDiscoveryResults(
	results []model.VideoSearchResult,
	limit int,
) []model.VideoSearchResult {
	filtered := make([]model.VideoSearchResult, 0, min(limit, len(results)))
	for _, result := range results {
		if provider.config.MinimumDuration > 0 &&
			time.Duration(result.Duration*float64(time.Second)) < provider.config.MinimumDuration {
			continue
		}
		if provider.config.ExcludeMusic && result.CategoryID == "10" {
			continue
		}
		if provider.config.ExcludeMusic && looksLikeMusic(result.Title) {
			continue
		}
		if containsExcludedTerm(
			result.Title+" "+result.Description,
			provider.config.ExcludedTerms,
		) {
			continue
		}
		filtered = append(filtered, result)
		if len(filtered) == limit {
			break
		}
	}
	return filtered
}

func containsExcludedTerm(value string, terms []string) bool {
	normalized := " " + normalizeWords(value) + " "
	for _, term := range terms {
		normalizedTerm := normalizeWords(term)
		if normalizedTerm != "" &&
			strings.Contains(normalized, " "+normalizedTerm+" ") {
			return true
		}
	}
	return false
}

func normalizeWords(value string) string {
	words := strings.FieldsFunc(strings.ToLower(value), func(character rune) bool {
		return (character < 'a' || character > 'z') &&
			(character < '0' || character > '9')
	})
	return strings.Join(words, " ")
}

func looksLikeMusic(title string) bool {
	lower := strings.ToLower(title)
	for _, marker := range []string{
		"official music video", "official audio", "lyrics", "lyric video",
		"full album", "music video", "karaoke",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func bestYouTubeThumbnail(thumbnails map[string]struct {
	URL string `json:"url"`
}) string {
	for _, name := range []string{"maxres", "standard", "high", "medium", "default"} {
		if thumbnail, ok := thumbnails[name]; ok {
			return thumbnail.URL
		}
	}
	return ""
}

func youtubeVideoID(link *url.URL) string {
	host := strings.ToLower(strings.TrimPrefix(link.Hostname(), "www."))
	if host == "youtu.be" {
		parts := strings.Split(strings.Trim(link.Path, "/"), "/")
		if len(parts) > 0 {
			return parts[0]
		}
		return ""
	}
	if id := link.Query().Get("v"); id != "" {
		return id
	}
	parts := strings.Split(strings.Trim(link.Path, "/"), "/")
	if len(parts) == 2 && (parts[0] == "shorts" || parts[0] == "embed") {
		return parts[1]
	}
	return ""
}

func parseYouTubeDuration(value string) time.Duration {
	matches := youtubeDurationPattern.FindStringSubmatch(value)
	if len(matches) != 5 {
		return 0
	}
	return time.Duration(
		parseMetric(matches[1])*24*int64(time.Hour) +
			parseMetric(matches[2])*int64(time.Hour) +
			parseMetric(matches[3])*int64(time.Minute) +
			parseMetric(matches[4])*int64(time.Second),
	)
}

func parseMetric(value string) int64 {
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return parsed
}
