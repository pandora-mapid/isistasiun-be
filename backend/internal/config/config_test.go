package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validConfig() *Config {
	return &Config{
		DatabaseURL:        "postgres://example",
		JWTSecret:          "a-secret-that-is-at-least-32-characters",
		JWTAccessTTLMins:   15,
		JWTRefreshTTLHours: 168,
		AuthRateLimitRPM:   10,
	}
}

func TestValidateAcceptsSecureAuthConfiguration(t *testing.T) {
	require.NoError(t, validConfig().Validate())
}

func TestValidateRejectsUnsafeAuthConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"missing database", func(c *Config) { c.DatabaseURL = "" }, "DATABASE_URL is required"},
		{"weak JWT secret", func(c *Config) { c.JWTSecret = "short" }, "JWT_SECRET must be at least 32 characters"},
		{"zero access ttl", func(c *Config) { c.JWTAccessTTLMins = 0 }, "JWT_ACCESS_TTL_MINUTES must be greater than zero"},
		{"zero refresh ttl", func(c *Config) { c.JWTRefreshTTLHours = 0 }, "JWT_REFRESH_TTL_HOURS must be greater than zero"},
		{"zero rate limit", func(c *Config) { c.AuthRateLimitRPM = 0 }, "AUTH_RATE_LIMIT_RPM must be greater than zero"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validConfig()
			tc.mutate(cfg)
			require.EqualError(t, cfg.Validate(), tc.want)
		})
	}
}
