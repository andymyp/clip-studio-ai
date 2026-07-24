package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	taskqueue "github.com/clipstudio-ai/clipstudio-ai/backend/internal/queue"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/repository"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/service"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Dependencies struct {
	DB           *gorm.DB
	Database     *repository.Connection
	Repositories *repository.Repositories
	Redis        *redis.Client
	AsynqClient  *asynq.Client
	Logger       *zap.Logger
	Auth         *service.AuthService
}

func Open(cfg Config) (*Dependencies, error) {
	if cfg.Environment == "production" && cfg.JWTSecret == developmentJWTSecret {
		return nil, errors.New("JWT_SECRET must be changed in production")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logLevel := logger.Warn
	if cfg.Environment == "development" {
		logLevel = logger.Info
	}

	appLogger, err := newLogger(cfg.Environment)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	connection, err := repository.Open(ctx, repository.Config{
		URL:             cfg.DatabaseURL,
		MaxOpenConns:    cfg.DBMaxOpen,
		MaxIdleConns:    cfg.DBMaxIdle,
		ConnMaxLifetime: cfg.DBMaxLife,
		LogLevel:        logLevel,
	})
	if err != nil {
		_ = appLogger.Sync()
		return nil, err
	}

	if cfg.AutoMigrate {
		appLogger.Info("auto migration enabled")
		if err := repository.Migrate(connection.GORM); err != nil {
			_ = connection.Close()
			_ = appLogger.Sync()
			return nil, err
		}
		appLogger.Info("database migration completed")
	} else {
		appLogger.Info("auto migration disabled")
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = connection.Close()
		_ = redisClient.Close()
		_ = appLogger.Sync()
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	asynqClient := taskqueue.NewClient(cfg.RedisAddr)
	repositories := repository.New(connection.GORM)
	authService, err := service.NewAuthService(
		repositories.Users,
		repository.NewRedisRefreshTokenStore(redisClient),
		service.AuthConfig{
			Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer,
			AccessTTL: cfg.AccessTokenTTL, RefreshTTL: cfg.RefreshTokenTTL,
			BcryptCost: cfg.BcryptCost,
		},
	)
	if err != nil {
		asynqClient.Close()
		_ = connection.Close()
		_ = redisClient.Close()
		_ = appLogger.Sync()
		return nil, fmt.Errorf("initialize authentication: %w", err)
	}
	return &Dependencies{
		DB:           connection.GORM,
		Database:     connection,
		Repositories: repositories,
		Redis:        redisClient,
		AsynqClient:  asynqClient,
		Logger:       appLogger,
		Auth:         authService,
	}, nil
}

func (d *Dependencies) Close() {
	d.AsynqClient.Close()
	_ = d.Redis.Close()
	_ = d.Database.Close()
	_ = d.Logger.Sync()
}

func newLogger(environment string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	if environment == "production" {
		config.EncoderConfig.EncodeTime = encodeLogTime
		return config.Build()
	}

	config = zap.NewDevelopmentConfig()
	config.Encoding = "console"
	config.EncoderConfig.EncodeTime = encodeLogTime
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return config.Build()
}

func encodeLogTime(value time.Time, encoder zapcore.PrimitiveArrayEncoder) {
	encoder.AppendString(value.Local().Format("2006-01-02 15:04"))
}
