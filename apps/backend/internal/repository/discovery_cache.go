package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/redis/go-redis/v9"
)

type RedisDiscoveryCache struct {
	client *redis.Client
}

func NewRedisDiscoveryCache(client *redis.Client) *RedisDiscoveryCache {
	return &RedisDiscoveryCache{client: client}
}

func (cache *RedisDiscoveryCache) Get(
	ctx context.Context,
	key string,
) ([]model.VideoSearchResult, bool) {
	payload, err := cache.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var results []model.VideoSearchResult
	if err := json.Unmarshal(payload, &results); err != nil {
		return nil, false
	}
	return results, true
}

func (cache *RedisDiscoveryCache) Set(
	ctx context.Context,
	key string,
	results []model.VideoSearchResult,
	ttl time.Duration,
) {
	payload, err := json.Marshal(results)
	if err != nil {
		return
	}
	_ = cache.client.Set(ctx, key, payload, ttl).Err()
}
