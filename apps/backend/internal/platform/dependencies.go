package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/config"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/infrastructure/database"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/infrastructure/database/migrations"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/infrastructure/persistence/gormrepo"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Dependencies struct {
	DB           *gorm.DB
	Database     *database.Connection
	Repositories *gormrepo.Repositories
	Redis        *redis.Client
	AsynqClient  *asynq.Client
}

func Open(cfg config.Config) (*Dependencies, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logLevel := logger.Warn
	if cfg.Environment == "development" {
		logLevel = logger.Info
	}

	connection, err := database.Open(ctx, database.Config{
		URL:             cfg.DatabaseURL,
		MaxOpenConns:    cfg.DBMaxOpen,
		MaxIdleConns:    cfg.DBMaxIdle,
		ConnMaxLifetime: cfg.DBMaxLife,
		LogLevel:        logLevel,
	})
	if err != nil {
		return nil, err
	}

	if cfg.AutoMigrate {
		if err := migrations.Up(connection.GORM); err != nil {
			_ = connection.Close()
			return nil, err
		}
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = connection.Close()
		_ = redisClient.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
	return &Dependencies{
		DB:           connection.GORM,
		Database:     connection,
		Repositories: gormrepo.New(connection.GORM),
		Redis:        redisClient,
		AsynqClient:  asynqClient,
	}, nil
}

func (d *Dependencies) Close() {
	d.AsynqClient.Close()
	_ = d.Redis.Close()
	_ = d.Database.Close()
}
