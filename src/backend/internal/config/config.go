package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const developmentJWTSecret = "dev-secret-change-in-production"

type Config struct {
	Port               string
	WorkerHealthPort   string
	DatabaseURL        string
	JWTSecret          string
	Environment        string
	CORSAllowedOrigins []string
	PublicBaseURL      string
	InvitationTTL      int
	JobPollInterval    int
	JobLeaseSeconds    int
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		WorkerHealthPort:   getEnv("WORKER_HEALTH_PORT", "8081"),
		DatabaseURL:        getEnv("DATABASE_URL", "sqlite://roomies.db"),
		JWTSecret:          getEnv("JWT_SECRET", developmentJWTSecret),
		Environment:        strings.ToLower(getEnv("APP_ENV", "development")),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:8081")),
		PublicBaseURL:      strings.TrimRight(getEnv("ROOMIES_PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
		InvitationTTL:      getEnvInt("INVITATION_TTL_HOURS", 168),
		JobPollInterval:    getEnvInt("JOB_POLL_INTERVAL_SECONDS", 2),
		JobLeaseSeconds:    getEnvInt("JOB_LEASE_SECONDS", 30),
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
	if c.Port == "" {
		return errors.New("PORT is required")
	}
	if c.WorkerHealthPort == "" {
		return errors.New("WORKER_HEALTH_PORT is required")
	}
	if c.PublicBaseURL == "" {
		return errors.New("ROOMIES_PUBLIC_BASE_URL is required")
	}
	if c.InvitationTTL <= 0 {
		return errors.New("INVITATION_TTL_HOURS must be positive")
	}
	if c.JobPollInterval <= 0 {
		return errors.New("JOB_POLL_INTERVAL_SECONDS must be positive")
	}
	if c.JobLeaseSeconds <= 0 {
		return errors.New("JOB_LEASE_SECONDS must be positive")
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

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}
