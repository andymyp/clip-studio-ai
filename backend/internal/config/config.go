package config

import "os"

type Config struct {
	Environment string
	Port        string
	DatabaseURL string
	RedisAddr   string
}

func Load() Config {
	return Config{
		Environment: env("APP_ENV", "development"),
		Port:        env("PORT", "8080"),
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
