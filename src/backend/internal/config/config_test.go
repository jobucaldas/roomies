package config

import "testing"

func TestConfigValidatesSMTPSettings(t *testing.T) {
	config := &Config{
		Environment:        "production",
		JWTSecret:          "a-production-secret-that-is-long-enough",
		CORSAllowedOrigins: []string{"https://roomies.example"},
		Port:               "8080",
		WorkerHealthPort:   "8081",
		PublicBaseURL:      "https://roomies.example",
		InvitationTTL:      168,
		JobPollInterval:    2,
		JobLeaseSeconds:    30,
		SMTPPort:           587,
		SMTPHost:           "smtp.example.com",
		SMTPFrom:           "Roomies <sender@example.com>",
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("expected valid SMTP settings: %v", err)
	}
	config.SMTPUsername = "user"
	if err := config.Validate(); err == nil {
		t.Fatal("expected username without password to be rejected")
	}
}

func TestProductionConfigRejectsWeakSecretAndWildcardCORS(t *testing.T) {
	config := &Config{
		Environment:        "production",
		JWTSecret:          developmentJWTSecret,
		CORSAllowedOrigins: []string{"https://roomies.example"},
		Port:               "8080",
		WorkerHealthPort:   "8081",
		PublicBaseURL:      "https://roomies.example",
		InvitationTTL:      168,
		JobPollInterval:    2,
		JobLeaseSeconds:    30,
	}
	if err := config.Validate(); err == nil {
		t.Fatal("expected development JWT secret to be rejected in production")
	}
	config.JWTSecret = "a-production-secret-that-is-long-enough"
	config.CORSAllowedOrigins = []string{"*"}
	if err := config.Validate(); err == nil {
		t.Fatal("expected wildcard production CORS origin to be rejected")
	}
	config.CORSAllowedOrigins = []string{"https://roomies.example"}
	if err := config.Validate(); err != nil {
		t.Fatalf("expected valid production configuration: %v", err)
	}
}
