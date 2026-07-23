package platform

import (
	"context"
	"fmt"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/config"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB          *gorm.DB
	Redis       *redis.Client
	AsynqClient *asynq.Client
}

func Open(cfg config.Config) (*Dependencies, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
	return &Dependencies{DB: db, Redis: redisClient, AsynqClient: asynqClient}, nil
}

func (d *Dependencies) Close() {
	d.AsynqClient.Close()
	_ = d.Redis.Close()
	if sqlDB, err := d.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
