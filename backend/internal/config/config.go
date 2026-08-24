package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv  string
	AppPort string

	DatabaseURL string

	JWTSecret          string
	JWTAccessTTLMins   int
	JWTRefreshTTLHours int

	PipelineServiceAPIKey string

	AIServiceURL            string
	AIServiceTimeoutSeconds int

	RateLimitRPM int

	CORSAllowedOrigins []string
}

func Load() *Config {
	// Ignore error: in containers env vars are injected directly, no .env file present.
	_ = godotenv.Load()

	return &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		JWTSecret:          getEnv("JWT_SECRET", ""),
		JWTAccessTTLMins:   getEnvInt("JWT_ACCESS_TTL_MINUTES", 15),
		JWTRefreshTTLHours: getEnvInt("JWT_REFRESH_TTL_HOURS", 168),

		PipelineServiceAPIKey: getEnv("PIPELINE_SERVICE_API_KEY", ""),

		AIServiceURL:            getEnv("AI_SERVICE_URL", "http://localhost:8000"),
		AIServiceTimeoutSeconds: getEnvInt("AI_SERVICE_TIMEOUT_SECONDS", 15),

		RateLimitRPM: getEnvInt("RATE_LIMIT_RPM", 60),

		CORSAllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ","),
	}
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
