package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Port        string
	DatabaseURL string
	RedisAddr   string
}

func Load() Config {
	// Support running from either the repository root or the backend directory.
	_ = godotenv.Load(".env.be")
	_ = godotenv.Load("../.env.be")

	return Config{
		Environment: env("APP_ENV", "development"),
		Port:        env("PORT", "3001"),
		DatabaseURL: env("DATABASE_URL", "postgres://clipstudio:clipstudio@localhost:5432/clipstudio?sslmode=disable"),
		RedisAddr:   env("REDIS_ADDR", "localhost:6379"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
