package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const refreshTokenPrefix = "auth:refresh:"

type RefreshTokenStore interface {
	Save(context.Context, string, uuid.UUID, time.Duration) error
	Consume(context.Context, string) (uuid.UUID, error)
	Delete(context.Context, string) error
}

type RedisRefreshTokenStore struct {
	client *redis.Client
}

func NewRedisRefreshTokenStore(client *redis.Client) *RedisRefreshTokenStore {
	return &RedisRefreshTokenStore{client: client}
}

func (store *RedisRefreshTokenStore) Save(
	ctx context.Context,
	tokenID string,
	userID uuid.UUID,
	ttl time.Duration,
) error {
	return store.client.Set(ctx, refreshTokenPrefix+tokenID, userID.String(), ttl).Err()
}

func (store *RedisRefreshTokenStore) Consume(ctx context.Context, tokenID string) (uuid.UUID, error) {
	value, err := store.client.GetDel(ctx, refreshTokenPrefix+tokenID).Result()
	if err != nil {
		if err == redis.Nil {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	userID, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (store *RedisRefreshTokenStore) Delete(ctx context.Context, tokenID string) error {
	return store.client.Del(ctx, refreshTokenPrefix+tokenID).Err()
}
