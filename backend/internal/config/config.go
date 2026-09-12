package config

import (
	"fmt"
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
	AuthRateLimitRPM   int
	AuthCookieDomain   string

	PipelineServiceAPIKey string

	AIServiceURL            string
	AIServiceTimeoutSeconds int

	RateLimitRPM   int
	TrustedProxies []string

	SummaryCacheTTLSeconds int

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
		AuthRateLimitRPM:   getEnvInt("AUTH_RATE_LIMIT_RPM", 10),
		AuthCookieDomain:   getEnv("AUTH_COOKIE_DOMAIN", ""),

		PipelineServiceAPIKey: getEnv("PIPELINE_SERVICE_API_KEY", ""),

		AIServiceURL:            getEnv("AI_SERVICE_URL", "http://localhost:8000"),
		AIServiceTimeoutSeconds: getEnvInt("AI_SERVICE_TIMEOUT_SECONDS", 15),

		RateLimitRPM:   getEnvInt("RATE_LIMIT_RPM", 60),
		TrustedProxies: getEnvList("TRUSTED_PROXIES", ""),

		// Station-summary rows only change when the batch pipeline re-runs
		// (../../../Context/02-BACKEND-SPEC.md §3.2), so a short shared TTL
		// cache is safe and cuts repeat DB reads from the Compare View.
		SummaryCacheTTLSeconds: getEnvInt("SUMMARY_CACHE_TTL_SECONDS", 300),

		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
	}
}

// Validate rejects configurations that would make authentication appear to
// work while using an empty/weak signing key or nonsensical lifetimes.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.JWTAccessTTLMins <= 0 {
		return fmt.Errorf("JWT_ACCESS_TTL_MINUTES must be greater than zero")
	}
	if c.JWTRefreshTTLHours <= 0 {
		return fmt.Errorf("JWT_REFRESH_TTL_HOURS must be greater than zero")
	}
	if c.AuthRateLimitRPM <= 0 {
		return fmt.Errorf("AUTH_RATE_LIMIT_RPM must be greater than zero")
	}
	return nil
}

func getEnvList(key, fallback string) []string {
	values := strings.Split(getEnv(key, fallback), ",")
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
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
