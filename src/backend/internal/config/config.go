package config

import (
	"errors"
	"os"
	"strings"
)

const developmentJWTSecret = "dev-secret-change-in-production"

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	Environment        string
	CORSAllowedOrigins []string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "sqlite://roomies.db"),
		JWTSecret:          getEnv("JWT_SECRET", developmentJWTSecret),
		Environment:        strings.ToLower(getEnv("APP_ENV", "development")),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:8081")),
	}
}

func (c *Config) Validate() error {
	if c.Environment == "production" {
		if c.JWTSecret == developmentJWTSecret || len(c.JWTSecret) < 32 {
			return errors.New("production JWT_SECRET must be set to at least 32 characters")
		}
		for _, origin := range c.CORSAllowedOrigins {
			if origin == "*" {
				return errors.New("wildcard CORS origin is not allowed in production")
			}
		}
	}
	if len(c.CORSAllowedOrigins) == 0 {
		return errors.New("at least one CORS_ALLOWED_ORIGINS value is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
