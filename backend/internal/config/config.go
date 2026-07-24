package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Port        string
	DatabaseURL string
	RedisAddr   string
	AutoMigrate bool
	DBMaxOpen   int
	DBMaxIdle   int
	DBMaxLife   time.Duration
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
		AutoMigrate: envBool("AUTO_MIGRATE", true),
		DBMaxOpen:   envInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdle:   envInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxLife:   envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
