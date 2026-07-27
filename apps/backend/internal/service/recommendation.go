package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RecommendationCollection struct {
	Videos            []model.RecommendedVideo
	KeywordsSucceeded []string
	KeywordErrors     map[string]string
	PopularSucceeded  bool
}

type RecommendationCollector interface {
	Collect(context.Context, []string, int) (RecommendationCollection, error)
}

type RecommendationStatus struct {
	State             string            `json:"state"`
	CatalogReady      bool              `json:"catalog_ready"`
	StartedAt         time.Time         `json:"started_at,omitempty"`
	CompletedAt       time.Time         `json:"completed_at,omitempty"`
	LastSuccessAt     time.Time         `json:"last_success_at,omitempty"`
	KeywordsRequested []string          `json:"keywords_requested,omitempty"`
	KeywordsSucceeded []string          `json:"keywords_succeeded,omitempty"`
	KeywordErrors     map[string]string `json:"keyword_errors,omitempty"`
	VideosCollected   int               `json:"videos_collected"`
	Error             string            `json:"error,omitempty"`
}

type RecommendationService struct {
	db       *gorm.DB
	redis    *redis.Client
	cacheTTL time.Duration
	limit    int
}

func NewRecommendationService(
	db *gorm.DB,
	redisClient *redis.Client,
	cacheTTL time.Duration,
	limit int,
) *RecommendationService {
	return &RecommendationService{
		db: db, redis: redisClient, cacheTTL: cacheTTL, limit: max(limit, 1),
	}
}

func (service *RecommendationService) Search(
	ctx context.Context,
	language string,
	category string,
) ([]model.VideoSearchResult, error) {
	language = normalizeLanguage(language)
	category = strings.ToLower(strings.TrimSpace(category))
	version, _ := service.redis.Get(ctx, "recommendations:version").Result()
	cacheKey := fmt.Sprintf("recommendations:v2:%s:%s:%s", version, language, category)
	if payload, err := service.redis.Get(ctx, cacheKey).Bytes(); err == nil {
		var cached []model.VideoSearchResult
		if json.Unmarshal(payload, &cached) == nil {
			return cached, nil
		}
	}

	var videos []model.RecommendedVideo
	err := service.db.WithContext(ctx).
		Table("recommended_videos").
		Select("recommended_videos.*, recommendation_categories.viral_score").
		Joins("JOIN recommendation_categories ON recommendation_categories.video_id = recommended_videos.id").
		Where(
			"recommendation_categories.language = ? AND recommendation_categories.category = ? AND recommended_videos.reusable = ?",
			language,
			category,
			true,
		).
		Order("recommendation_categories.viral_score DESC, recommended_videos.published_at DESC").
		Limit(service.limit).
		Scan(&videos).Error
	if err != nil {
		return nil, fmt.Errorf("query recommendations: %w", err)
	}
	results := make([]model.VideoSearchResult, 0, len(videos))
	for _, video := range videos {
		results = append(results, recommendationResult(video))
	}
	if payload, err := json.Marshal(results); err == nil {
		_ = service.redis.Set(ctx, cacheKey, payload, service.cacheTTL).Err()
	}
	return results, nil
}

func (service *RecommendationService) Status(
	ctx context.Context,
) (RecommendationStatus, error) {
	catalogReady, err := service.HasCatalog(ctx)
	if err != nil {
		return RecommendationStatus{}, err
	}
	payload, err := service.redis.Get(ctx, "recommendations:scheduler:status").Bytes()
	if err == redis.Nil {
		return RecommendationStatus{State: "waiting", CatalogReady: catalogReady}, nil
	}
	if err != nil {
		return RecommendationStatus{}, fmt.Errorf("load recommendation status: %w", err)
	}
	var status RecommendationStatus
	if err := json.Unmarshal(payload, &status); err != nil {
		return RecommendationStatus{}, fmt.Errorf("decode recommendation status: %w", err)
	}
	status.CatalogReady = catalogReady
	return status, nil
}

func (service *RecommendationService) HasCatalog(ctx context.Context) (bool, error) {
	var count int64
	if err := service.db.WithContext(ctx).
		Model(&model.RecommendationCategory{}).
		Joins("JOIN recommended_videos ON recommended_videos.id = recommendation_categories.video_id").
		Where("recommended_videos.reusable = ?", true).
		Limit(1).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check recommendation catalog: %w", err)
	}
	return count > 0, nil
}

func (service *RecommendationService) setStatus(
	ctx context.Context,
	status RecommendationStatus,
) {
	payload, err := json.Marshal(status)
	if err == nil {
		_ = service.redis.Set(ctx, "recommendations:scheduler:status", payload, 0).Err()
	}
}

func (service *RecommendationService) Save(
	ctx context.Context,
	videos []model.RecommendedVideo,
) error {
	videos = deduplicateRecommendations(videos)
	if len(videos) == 0 {
		return service.cleanup(ctx, time.Now().UTC())
	}
	now := time.Now().UTC()
	channelGrowth, snapshots, err := service.channelGrowth(ctx, videos, now)
	if err != nil {
		return err
	}

	ids := make([]string, 0, len(videos))
	for _, video := range videos {
		ids = append(ids, video.ExternalID)
	}
	var existing []model.RecommendedVideo
	if err := service.db.WithContext(ctx).
		Where("external_id IN ?", ids).
		Find(&existing).Error; err != nil {
		return fmt.Errorf("load existing recommendations: %w", err)
	}
	previous := make(map[string]model.RecommendedVideo, len(existing))
	externalIDByVideoID := make(map[uuid.UUID]string, len(existing))
	for _, video := range existing {
		previous[video.ExternalID] = video
		externalIDByVideoID[video.ID] = video.ExternalID
	}
	if len(existing) > 0 {
		videoIDs := make([]uuid.UUID, 0, len(existing))
		for _, video := range existing {
			videoIDs = append(videoIDs, video.ID)
		}
		var categories []model.RecommendationCategory
		if err := service.db.WithContext(ctx).
			Where("video_id IN ?", videoIDs).
			Find(&categories).Error; err != nil {
			return fmt.Errorf("load recommendation categories: %w", err)
		}
		existingCategories := make(map[string][]string)
		for _, category := range categories {
			externalID := externalIDByVideoID[category.VideoID]
			existingCategories[externalID] = append(
				existingCategories[externalID],
				category.Category,
			)
		}
		for index := range videos {
			videos[index].Categories = normalizedCategories(
				append(videos[index].Categories, existingCategories[videos[index].ExternalID]...),
				"",
			)
		}
	}
	scoreRecommendations(videos, channelGrowth, now)
	for index := range videos {
		video := &videos[index]
		video.EnsureID()
		video.CollectedAt = now
		if old, ok := previous[video.ExternalID]; ok {
			video.ID = old.ID
			video.CreatedAt = old.CreatedAt
		}
	}

	err = service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "external_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"platform", "language", "category_id", "title", "description",
				"channel_id", "channel_title", "you_tube_username", "url",
				"embed_url", "thumbnail", "duration", "views", "likes",
				"comments", "subscribers", "published_at", "collected_at",
				"viral_score", "license", "reusable", "updated_at",
			}),
		}).Create(&videos).Error; err != nil {
			return err
		}
		var persisted []model.RecommendedVideo
		if err := tx.Where("external_id IN ?", ids).Find(&persisted).Error; err != nil {
			return err
		}
		persistedIDs := make(map[string]uuid.UUID, len(persisted))
		for _, video := range persisted {
			persistedIDs[video.ExternalID] = video.ID
		}
		for index := range videos {
			videos[index].ID = persistedIDs[videos[index].ExternalID]
		}
		for _, video := range videos {
			if err := tx.Where("video_id = ?", video.ID).
				Delete(&model.RecommendationCategory{}).Error; err != nil {
				return err
			}
			categories := make([]model.RecommendationCategory, 0, len(video.Categories))
			for _, category := range video.Categories {
				categories = append(categories, model.RecommendationCategory{
					VideoID: video.ID, Category: category, Language: video.Language,
					ViralScore: video.CategoryScores[category],
				})
			}
			if len(categories) > 0 {
				if err := tx.Create(&categories).Error; err != nil {
					return err
				}
			}
		}
		if len(snapshots) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
				Create(&snapshots).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("save recommendations: %w", err)
	}
	if err := service.cleanup(ctx, now); err != nil {
		return err
	}
	_ = service.redis.Incr(ctx, "recommendations:version").Err()
	return nil
}

func (service *RecommendationService) channelGrowth(
	ctx context.Context,
	videos []model.RecommendedVideo,
	now time.Time,
) (map[string]float64, []model.ChannelMetricSnapshot, error) {
	channelIDs := make([]string, 0, len(videos))
	current := make(map[string]int64)
	for _, video := range videos {
		if video.ChannelID != "" {
			if _, exists := current[video.ChannelID]; !exists {
				channelIDs = append(channelIDs, video.ChannelID)
			}
			current[video.ChannelID] = video.Subscribers
		}
	}
	var history []model.ChannelMetricSnapshot
	if len(channelIDs) > 0 {
		if err := service.db.WithContext(ctx).
			Where("channel_id IN ? AND captured_at >= ?", channelIDs, now.Add(-7*24*time.Hour)).
			Order("captured_at DESC").
			Find(&history).Error; err != nil {
			return nil, nil, fmt.Errorf("load channel snapshots: %w", err)
		}
	}
	latest := make(map[string]model.ChannelMetricSnapshot)
	for _, snapshot := range history {
		if _, exists := latest[snapshot.ChannelID]; !exists {
			latest[snapshot.ChannelID] = snapshot
		}
	}
	growth := make(map[string]float64, len(current))
	snapshots := make([]model.ChannelMetricSnapshot, 0, len(current))
	capturedAt := now.Truncate(time.Minute)
	for channelID, subscribers := range current {
		if old, exists := latest[channelID]; exists &&
			old.Subscribers > 0 && subscribers > old.Subscribers {
			elapsedHours := math.Max(now.Sub(old.CapturedAt).Hours(), 0.5)
			growth[channelID] = (float64(subscribers-old.Subscribers) /
				float64(old.Subscribers)) / elapsedHours
		}
		snapshot := model.ChannelMetricSnapshot{
			ChannelID: channelID, Subscribers: subscribers, CapturedAt: capturedAt,
		}
		snapshot.EnsureID()
		snapshots = append(snapshots, snapshot)
	}
	return growth, snapshots, nil
}

func (service *RecommendationService) cleanup(ctx context.Context, now time.Time) error {
	return service.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("reusable = ?", false).
			Delete(&model.RecommendedVideo{}).Error; err != nil {
			return fmt.Errorf("delete non-reusable recommendations: %w", err)
		}
		if err := tx.Where("published_at < ?", now.Add(-7*24*time.Hour)).
			Delete(&model.RecommendedVideo{}).Error; err != nil {
			return fmt.Errorf("delete expired recommendations: %w", err)
		}
		if err := tx.Where("captured_at < ?", now.Add(-30*24*time.Hour)).
			Delete(&model.ChannelMetricSnapshot{}).Error; err != nil {
			return fmt.Errorf("delete expired channel snapshots: %w", err)
		}
		return nil
	})
}

func deduplicateRecommendations(
	videos []model.RecommendedVideo,
) []model.RecommendedVideo {
	unique := make([]model.RecommendedVideo, 0, len(videos))
	indexByID := make(map[string]int, len(videos))
	for _, video := range videos {
		video.ExternalID = strings.TrimSpace(video.ExternalID)
		if video.ExternalID == "" {
			continue
		}
		video.Categories = normalizedCategories(video.Categories, video.Category)
		if index, exists := indexByID[video.ExternalID]; exists {
			merged := append(unique[index].Categories, video.Categories...)
			video.Categories = normalizedCategories(merged, "")
			unique[index] = video
			continue
		}
		indexByID[video.ExternalID] = len(unique)
		unique = append(unique, video)
	}
	return unique
}

func normalizedCategories(categories []string, legacy string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(categories)+1)
	categories = append(categories, legacy)
	for _, category := range categories {
		category = strings.ToLower(strings.TrimSpace(category))
		if category == "" {
			continue
		}
		if _, exists := seen[category]; !exists {
			seen[category] = struct{}{}
			result = append(result, category)
		}
	}
	sort.Strings(result)
	return result
}

type scoreFactors struct {
	viewVelocity, likeRate, commentRate, freshness, channelGrowth, subscriberRatio float64
}

func scoreRecommendations(
	videos []model.RecommendedVideo,
	channelGrowth map[string]float64,
	now time.Time,
) {
	factors := make([]scoreFactors, len(videos))
	groups := make(map[string][]int)
	for index := range videos {
		video := &videos[index]
		ageHours := math.Max(now.Sub(video.PublishedAt).Hours(), 1)
		factors[index] = scoreFactors{
			viewVelocity:    float64(video.Views) / ageHours,
			likeRate:        ratio(video.Likes, video.Views),
			commentRate:     ratio(video.Comments, video.Views),
			freshness:       math.Max(0, 24-ageHours),
			channelGrowth:   channelGrowth[video.ChannelID],
			subscriberRatio: ratio(video.Views, video.Subscribers),
		}
		video.CategoryScores = make(map[string]float64, len(video.Categories))
		for _, category := range video.Categories {
			key := video.Language + "\x00" + category
			groups[key] = append(groups[key], index)
		}
	}
	for _, indexes := range groups {
		for _, index := range indexes {
			score := 0.35*percentile(factors, indexes, index, func(f scoreFactors) float64 { return f.viewVelocity }) +
				0.15*percentile(factors, indexes, index, func(f scoreFactors) float64 { return f.likeRate }) +
				0.15*percentile(factors, indexes, index, func(f scoreFactors) float64 { return f.commentRate }) +
				0.15*percentile(factors, indexes, index, func(f scoreFactors) float64 { return f.freshness }) +
				0.10*percentile(factors, indexes, index, func(f scoreFactors) float64 { return f.channelGrowth }) +
				0.10*percentile(factors, indexes, index, func(f scoreFactors) float64 { return f.subscriberRatio })
			for _, category := range videos[index].Categories {
				key := videos[index].Language + "\x00" + category
				if sameIndexes(groups[key], indexes) {
					videos[index].CategoryScores[category] = math.Round(score*100) / 100
					videos[index].ViralScore = math.Max(videos[index].ViralScore, score)
				}
			}
		}
	}
}

func percentile(
	factors []scoreFactors,
	indexes []int,
	target int,
	value func(scoreFactors) float64,
) float64 {
	if len(indexes) == 1 {
		if value(factors[target]) > 0 {
			return 50
		}
		return 0
	}
	targetValue := value(factors[target])
	lower := 0
	equal := 0
	for _, index := range indexes {
		candidate := value(factors[index])
		if candidate < targetValue {
			lower++
		} else if candidate == targetValue {
			equal++
		}
	}
	rank := float64(lower) + float64(equal-1)/2
	return rank / float64(len(indexes)-1) * 100
}

func sameIndexes(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func ratio(numerator, denominator int64) float64 {
	if numerator <= 0 || denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func normalizeLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if index := strings.Index(value, "-"); index >= 0 {
		value = value[:index]
	}
	return value
}

func recommendationResult(video model.RecommendedVideo) model.VideoSearchResult {
	return model.VideoSearchResult{
		ExternalID: video.ExternalID, Platform: video.Platform,
		CategoryID: video.CategoryID, Title: video.Title,
		ChannelID: video.ChannelID, ChannelTitle: video.ChannelTitle,
		YouTubeUsername: video.YouTubeUsername, URL: video.URL,
		EmbedURL: video.EmbedURL, Thumbnail: video.Thumbnail,
		Duration: video.Duration, Views: video.Views, Likes: video.Likes,
		Comments: video.Comments, Score: int64(math.Round(video.ViralScore)),
		License: video.License, Reusable: video.Reusable,
	}
}

type RecommendationScheduler struct {
	collector    RecommendationCollector
	service      *RecommendationService
	keywords     []string
	interval     time.Duration
	batchSize    int
	keywordsEach int
	logger       *zap.Logger
	cancel       context.CancelFunc
	done         chan struct{}
	once         sync.Once
}

func NewRecommendationScheduler(
	collector RecommendationCollector,
	service *RecommendationService,
	keywords []string,
	interval time.Duration,
	batchSize int,
	keywordsEach int,
	logger *zap.Logger,
) *RecommendationScheduler {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &RecommendationScheduler{
		collector: collector, service: service, keywords: normalizedCategories(keywords, ""),
		interval: interval, batchSize: batchSize, keywordsEach: max(keywordsEach, 1),
		logger: logger,
	}
}

func (scheduler *RecommendationScheduler) Start() {
	if scheduler == nil || scheduler.collector == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	scheduler.cancel = cancel
	scheduler.done = make(chan struct{})
	hasCatalog, err := scheduler.service.HasCatalog(ctx)
	if err != nil {
		scheduler.logger.Warn(
			"could not inspect recommendation catalog; running warm-up",
			zap.Error(err),
		)
	}
	if !hasCatalog {
		scheduler.logger.Info(
			"recommendation catalog is empty; running startup warm-up",
		)
	}
	go func() {
		defer close(scheduler.done)
		if !hasCatalog {
			// Start immediately but leave the status endpoint available so the
			// frontend can report warm-up progress.
			scheduler.run(ctx)
		}
		ticker := time.NewTicker(scheduler.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				scheduler.run(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (scheduler *RecommendationScheduler) Stop() {
	if scheduler == nil || scheduler.cancel == nil {
		return
	}
	scheduler.once.Do(func() {
		scheduler.cancel()
		<-scheduler.done
	})
}

func (scheduler *RecommendationScheduler) rotatedKeywords(ctx context.Context) []string {
	if len(scheduler.keywords) <= scheduler.keywordsEach {
		return scheduler.keywords
	}
	cycle, err := scheduler.service.redis.Incr(ctx, "recommendations:scheduler:cycle").Result()
	if err != nil {
		cycle = 1
	}
	start := int((cycle-1)*int64(scheduler.keywordsEach)) % len(scheduler.keywords)
	result := make([]string, 0, scheduler.keywordsEach)
	for offset := 0; offset < scheduler.keywordsEach; offset++ {
		result = append(result, scheduler.keywords[(start+offset)%len(scheduler.keywords)])
	}
	return result
}

func (scheduler *RecommendationScheduler) run(ctx context.Context) {
	lockToken := uuid.NewString()
	locked, err := scheduler.service.redis.SetNX(
		ctx, "recommendations:scheduler:lock", lockToken,
		max(scheduler.interval-time.Minute, 5*time.Minute),
	).Result()
	if err != nil || !locked {
		if err != nil {
			scheduler.logger.Warn("recommendation scheduler lock unavailable", zap.Error(err))
		}
		return
	}
	defer scheduler.releaseLock(lockToken)

	keywords := scheduler.rotatedKeywords(ctx)
	previousStatus, _ := scheduler.service.Status(ctx)
	status := RecommendationStatus{
		State: "running", StartedAt: time.Now().UTC(),
		LastSuccessAt: previousStatus.LastSuccessAt, KeywordsRequested: keywords,
	}
	scheduler.service.setStatus(ctx, status)
	if err := scheduler.service.cleanup(ctx, time.Now().UTC()); err != nil {
		status.State = "failed"
		status.CompletedAt = time.Now().UTC()
		status.Error = err.Error()
		scheduler.service.setStatus(context.Background(), status)
		if ctx.Err() == nil {
			scheduler.logger.Error("recommendation cleanup failed", zap.Error(err))
		}
		return
	}
	collection, collectErr := scheduler.collector.Collect(ctx, keywords, scheduler.batchSize)
	status.CompletedAt = time.Now().UTC()
	status.KeywordsSucceeded = collection.KeywordsSucceeded
	status.KeywordErrors = collection.KeywordErrors
	status.VideosCollected = len(collection.Videos)
	if collectErr == nil || len(collection.Videos) > 0 {
		if err := scheduler.service.Save(ctx, collection.Videos); err != nil {
			collectErr = err
		}
	}
	if collectErr != nil {
		status.State = "failed"
		status.Error = collectErr.Error()
		if ctx.Err() == nil {
			scheduler.logger.Error("recommendation refresh failed", zap.Error(collectErr))
		}
	} else {
		status.State = "completed"
		status.LastSuccessAt = status.CompletedAt
		scheduler.logger.Info(
			"recommendations refreshed",
			zap.Int("videos", len(collection.Videos)),
			zap.Strings("keywords", collection.KeywordsSucceeded),
			zap.Int("keyword_failures", len(collection.KeywordErrors)),
		)
	}
	scheduler.service.setStatus(context.Background(), status)
}

func (scheduler *RecommendationScheduler) releaseLock(token string) {
	const script = `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		end
		return 0
	`
	_ = scheduler.service.redis.Eval(
		context.Background(), script,
		[]string{"recommendations:scheduler:lock"}, token,
	).Err()
}
