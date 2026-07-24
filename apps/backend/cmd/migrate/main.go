package main

import (
	"context"
	"log"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/config"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/infrastructure/database"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/infrastructure/database/migrations"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connection, err := database.Open(ctx, database.Config{
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

	if err := migrations.Up(connection.GORM); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	log.Println("database migrations completed")
}
