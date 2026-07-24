package main

import (
	"context"
	"log"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/api"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/repository"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := api.LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connection, err := repository.Open(ctx, repository.Config{
		URL:             cfg.DatabaseURL,
		MaxOpenConns:    cfg.DBMaxOpen,
		MaxIdleConns:    cfg.DBMaxIdle,
		ConnMaxLifetime: cfg.DBMaxLife,
		LogLevel:        logger.Warn,
	})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer connection.Close()

	if err := repository.Migrate(connection.GORM); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	log.Println("database migrations completed")
}
