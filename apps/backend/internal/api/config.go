package api

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const developmentJWTSecret = "development-only-change-this-secret"

type Config struct {
	Environment           string
	Port                  string
	DatabaseURL           string
	RedisAddr             string
	AsynqQueue            string
	AutoMigrate           bool
	DBMaxOpen             int
	DBMaxIdle             int
	DBMaxLife             time.Duration
	ShutdownTimeout       time.Duration
	JWTSecret             string
	JWTIssuer             string
	AccessTokenTTL        time.Duration
	RefreshTokenTTL       time.Duration
	BcryptCost            int
	EnableYouTubeAPI      bool
	YouTubeAPIKey         string
	YouTubeRegion         string
	YouTubeLanguage       string
	YouTubeDiscoveryQuery string
	YouTubeReusableOnly   bool
	YouTubeExcludeMusic   bool
	YouTubeExcludedTerms  string
	YouTubeDiscoveryDays  int
	YouTubeMinDuration    int
	EnableRedditAPI       bool
	RedditClientID        string
	RedditSecret          string
	RedditUserAgent       string
	DiscoveryMinResults   int
	DiscoveryLimit        int
	DiscoveryTimeout      time.Duration
	DiscoveryCacheTTL     time.Duration
	WorkerAPIURL          string
}

func LoadConfig() Config {
	// Support running from either the repository root or the backend directory.
	_ = godotenv.Load(".env.be")
	_ = godotenv.Load("../.env.be")
	_ = godotenv.Load("../../.env.be")

	return Config{
		Environment:      env("APP_ENV", "development"),
		Port:             env("PORT", "3001"),
		DatabaseURL:      env("DATABASE_URL", "postgres://clipstudio:clipstudio@localhost:5432/clipstudio?sslmode=disable"),
		RedisAddr:        env("REDIS_ADDR", "localhost:6379"),
		AsynqQueue:       env("ASYNQ_QUEUE", "default"),
		AutoMigrate:      envBool("AUTO_MIGRATE", true),
		DBMaxOpen:        envInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdle:        envInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxLife:        envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		ShutdownTimeout:  envDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		JWTSecret:        env("JWT_SECRET", developmentJWTSecret),
		JWTIssuer:        env("JWT_ISSUER", "clipstudio-ai"),
		AccessTokenTTL:   envDuration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTokenTTL:  envDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		BcryptCost:       envInt("BCRYPT_COST", 12),
		EnableYouTubeAPI: envBool("ENABLE_YOUTUBE_API", true),
		YouTubeAPIKey:    env("YOUTUBE_API_KEY", ""),
		YouTubeRegion:    env("YOUTUBE_REGION", ""),
		YouTubeLanguage:  env("YOUTUBE_LANGUAGE", "en"),
		YouTubeDiscoveryQuery: env(
			"YOUTUBE_DISCOVERY_QUERY",
			"podcast|interview|education|business|technology|science|story|debate|speech|documentary",
		),
		YouTubeReusableOnly: envBool("YOUTUBE_REUSABLE_ONLY", true),
		YouTubeExcludeMusic: envBool("YOUTUBE_EXCLUDE_MUSIC", true),
		YouTubeExcludedTerms: env(
			"YOUTUBE_EXCLUDED_TERMS",
			"religion,religious,faith,church,christian,muslim,islam,hindu,politics,political,election,war,weapon,gun,violence,violent,crime,murder,adult,sexual,gambling,casino,drug",
		),
		YouTubeDiscoveryDays: envInt("YOUTUBE_DISCOVERY_DAYS", 30),
		YouTubeMinDuration:   envInt("YOUTUBE_MIN_DURATION_SECONDS", 180),
		EnableRedditAPI:      envBool("ENABLE_REDDIT_API", false),
		RedditClientID:       env("REDDIT_CLIENT_ID", ""),
		RedditSecret:         env("REDDIT_CLIENT_SECRET", ""),
		RedditUserAgent:      env("REDDIT_USER_AGENT", "web:clipstudio-ai:v0.1.0"),
		DiscoveryMinResults:  envInt("DISCOVERY_MIN_RESULTS", 10),
		DiscoveryLimit:       envInt("DISCOVERY_LIMIT", 20),
		DiscoveryTimeout:     envDuration("DISCOVERY_TIMEOUT", 12*time.Second),
		DiscoveryCacheTTL:    envDuration("DISCOVERY_CACHE_TTL", 6*time.Hour),
		WorkerAPIURL:         env("WORKER_API_URL", "http://localhost:3002"),
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
