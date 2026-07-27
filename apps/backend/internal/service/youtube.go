package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
)

const (
	youtubeAPIBase  = "https://www.googleapis.com/youtube/v3"
	youtubeAPIDelay = 1 * time.Second
)

var youtubeDurationPattern = regexp.MustCompile(`^P(?:(\d+)D)?T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)

type YouTubeProvider struct {
	apiKey string
	client *http.Client
	config YouTubeDiscoveryConfig

	requestMu   sync.Mutex
	lastRequest time.Time
}

type YouTubeDiscoveryConfig struct {
	Region string
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

func youtubePageSize(limit int) int {
	return min(max(limit, 1), 50)
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
	if !results[0].Reusable || results[0].License != "creativeCommon" {
		return nil, ErrNonReusableVideo
	}
	return &results[0], nil
}

func (provider *YouTubeProvider) videos(
	ctx context.Context,
	ids []string,
) ([]model.VideoSearchResult, error) {
	return provider.videoDetails(ctx, ids, true)
}

func (provider *YouTubeProvider) videoDetails(
	ctx context.Context,
	ids []string,
	loadHandles bool,
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
	if loadHandles {
		handles, err := provider.channelHandles(ctx, channelIDs)
		if err == nil {
			for index := range results {
				results[index].YouTubeUsername = handles[results[index].ChannelID]
			}
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
	const maximumAttempts = 3
	for attempt := 0; attempt < maximumAttempts; attempt++ {
		if err := provider.waitForAPISlot(ctx); err != nil {
			return err
		}
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
		if response.StatusCode == http.StatusOK {
			err = json.NewDecoder(response.Body).Decode(target)
			response.Body.Close()
			return err
		}
		_, _ = io.Copy(io.Discard, response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusTooManyRequests ||
			attempt == maximumAttempts-1 {
			if response.StatusCode == http.StatusTooManyRequests {
				return fmt.Errorf(
					"%w: youtube API returned %s",
					ErrDiscoveryRateLimited,
					response.Status,
				)
			}
			return fmt.Errorf("youtube API returned %s", response.Status)
		}
		if err := waitForYouTubeRequest(
			ctx,
			youtubeRetryDelay(response.Header.Get("Retry-After"), attempt),
		); err != nil {
			return err
		}
	}
	return fmt.Errorf("youtube API retry limit reached")
}

func (provider *YouTubeProvider) waitForAPISlot(ctx context.Context) error {
	provider.requestMu.Lock()
	defer provider.requestMu.Unlock()

	delay := time.Until(provider.lastRequest.Add(youtubeAPIDelay))
	if delay > 0 {
		if err := waitForYouTubeRequest(ctx, delay); err != nil {
			return err
		}
	}
	provider.lastRequest = time.Now()
	return nil
}

func youtubeRetryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil &&
		seconds > 0 {
		return min(time.Duration(seconds)*time.Second, 5*time.Second)
	}
	return time.Duration(1<<attempt) * time.Second
}

func waitForYouTubeRequest(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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
		Title                string `json:"title"`
		Description          string `json:"description"`
		CategoryID           string `json:"categoryId"`
		ChannelID            string `json:"channelId"`
		ChannelTitle         string `json:"channelTitle"`
		PublishedAt          string `json:"publishedAt"`
		DefaultLanguage      string `json:"defaultLanguage"`
		DefaultAudioLanguage string `json:"defaultAudioLanguage"`
		Thumbnails           map[string]struct {
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
			Language: firstNonEmpty(
				item.Snippet.DefaultAudioLanguage,
				item.Snippet.DefaultLanguage,
			),
			PublishedAt: parseYouTubeTime(item.Snippet.PublishedAt),
			URL:         "https://www.youtube.com/watch?v=" + item.ID,
			EmbedURL:    "https://www.youtube.com/embed/" + item.ID,
			Thumbnail:   bestYouTubeThumbnail(item.Snippet.Thumbnails),
			Duration:    parseYouTubeDuration(item.ContentDetails.Duration).Seconds(),
			Views:       parseMetric(item.Statistics.ViewCount),
			Likes:       parseMetric(item.Statistics.LikeCount),
			Comments:    parseMetric(item.Statistics.CommentCount),
			License:     item.Status.License,
			Reusable:    item.Status.License == "creativeCommon",
		})
	}
	return results
}

func parseYouTubeTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
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

func (provider *YouTubeProvider) Collect(
	ctx context.Context,
	keywords []string,
	limit int,
) (RecommendationCollection, error) {
	type candidate struct {
		id         string
		categories map[string]struct{}
	}
	candidates := make(map[string]candidate)
	collection := RecommendationCollection{
		KeywordErrors: make(map[string]string),
	}

	popularQuery := url.Values{
		"part": {"id"}, "chart": {"mostPopular"},
		"maxResults": {strconv.Itoa(youtubePageSize(limit))},
		"key":        {provider.apiKey},
	}
	if provider.config.Region != "" {
		popularQuery.Set("regionCode", provider.config.Region)
	}
	var popular youtubeVideosResponse
	if err := provider.get(ctx, "/videos", popularQuery, &popular); err != nil {
		collection.KeywordErrors["trending"] = err.Error()
	} else {
		collection.PopularSucceeded = true
		for _, item := range popular.Items {
			candidates[item.ID] = candidate{
				id: item.ID, categories: map[string]struct{}{"trending": {}},
			}
		}
	}

	publishedAfter := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	for _, rawKeyword := range keywords {
		keyword := strings.ToLower(strings.TrimSpace(rawKeyword))
		if keyword == "" {
			continue
		}
		query := url.Values{
			"part":            {"snippet"},
			"type":            {"video"},
			"order":           {"date"},
			"publishedAfter":  {publishedAfter},
			"videoDuration":   {"long"},
			"videoEmbeddable": {"true"},
			"videoLicense":    {"creativeCommon"},
			"maxResults":      {strconv.Itoa(youtubePageSize(limit))},
			"q":               {keyword},
			"safeSearch":      {"strict"},
			"key":             {provider.apiKey},
		}
		var response youtubeSearchResponse
		if err := provider.get(ctx, "/search", query, &response); err != nil {
			collection.KeywordErrors[keyword] = err.Error()
			continue
		}
		collection.KeywordsSucceeded = append(collection.KeywordsSucceeded, keyword)
		for _, item := range response.Items {
			if item.ID.VideoID == "" {
				continue
			}
			value, exists := candidates[item.ID.VideoID]
			if !exists {
				value = candidate{
					id: item.ID.VideoID, categories: make(map[string]struct{}),
				}
			}
			value.categories[keyword] = struct{}{}
			candidates[item.ID.VideoID] = value
		}
	}

	ids := make([]string, 0, len(candidates))
	for id := range candidates {
		ids = append(ids, id)
	}
	results := make([]model.VideoSearchResult, 0, len(ids))
	for start := 0; start < len(ids); start += 50 {
		end := min(start+50, len(ids))
		batch, err := provider.videoDetails(ctx, ids[start:end], false)
		if err != nil {
			collection.KeywordErrors["enrichment"] = err.Error()
			continue
		}
		results = append(results, batch...)
	}
	results = creativeCommonsOnly(results)
	if len(results) == 0 {
		return collection, fmt.Errorf("no Creative Commons recommendation videos were found")
	}
	channels, err := provider.recommendationChannels(ctx, results)
	if err != nil {
		collection.KeywordErrors["channels"] = err.Error()
		channels = make(map[string]recommendationChannel)
	}

	now := time.Now().UTC()
	recommendations := make([]model.RecommendedVideo, 0, len(results))
	for _, result := range results {
		language := normalizeLanguage(result.Language)
		if language == "" || result.PublishedAt.IsZero() {
			continue
		}
		channel := channels[result.ChannelID]
		categorySet := candidates[result.ExternalID].categories
		categories := make([]string, 0, len(categorySet))
		for category := range categorySet {
			categories = append(categories, category)
		}
		recommendations = append(recommendations, model.RecommendedVideo{
			ExternalID: result.ExternalID, Platform: "youtube",
			Categories: categories, Language: language, CategoryID: result.CategoryID,
			Title: result.Title, Description: result.Description,
			ChannelID: result.ChannelID, ChannelTitle: result.ChannelTitle,
			YouTubeUsername: channel.handle, URL: result.URL,
			EmbedURL: result.EmbedURL, Thumbnail: result.Thumbnail,
			Duration: result.Duration, Views: result.Views, Likes: result.Likes,
			Comments: result.Comments, Subscribers: channel.subscribers,
			PublishedAt: result.PublishedAt, CollectedAt: now,
			License: result.License, Reusable: result.Reusable,
		})
	}
	collection.Videos = recommendations
	return collection, nil
}

func creativeCommonsOnly(
	results []model.VideoSearchResult,
) []model.VideoSearchResult {
	filtered := results[:0]
	for _, result := range results {
		if result.License == "creativeCommon" && result.Reusable {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

type recommendationChannel struct {
	handle      string
	subscribers int64
}

func (provider *YouTubeProvider) recommendationChannels(
	ctx context.Context,
	videos []model.VideoSearchResult,
) (map[string]recommendationChannel, error) {
	unique := make(map[string]struct{})
	for _, video := range videos {
		if video.ChannelID != "" {
			unique[video.ChannelID] = struct{}{}
		}
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	channels := make(map[string]recommendationChannel, len(ids))
	for start := 0; start < len(ids); start += 50 {
		end := min(start+50, len(ids))
		query := url.Values{
			"part": {"snippet,statistics"},
			"id":   {strings.Join(ids[start:end], ",")},
			"key":  {provider.apiKey},
		}
		var response struct {
			Items []struct {
				ID      string `json:"id"`
				Snippet struct {
					CustomURL string `json:"customUrl"`
				} `json:"snippet"`
				Statistics struct {
					SubscriberCount string `json:"subscriberCount"`
				} `json:"statistics"`
			} `json:"items"`
		}
		if err := provider.get(ctx, "/channels", query, &response); err != nil {
			return nil, fmt.Errorf("load recommendation channels: %w", err)
		}
		for _, item := range response.Items {
			handle := strings.TrimSpace(item.Snippet.CustomURL)
			if handle != "" && !strings.HasPrefix(handle, "@") {
				handle = "@" + handle
			}
			channels[item.ID] = recommendationChannel{
				handle: handle, subscribers: parseMetric(item.Statistics.SubscriberCount),
			}
		}
	}
	return channels, nil
}
