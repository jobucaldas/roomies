package config

import (
	"encoding/base64"
	"testing"
)

func testDeliveryKey() string { return base64.StdEncoding.EncodeToString(make([]byte, 32)) }

func TestConfigValidatesSMTPSettings(t *testing.T) {
	config := &Config{
		Environment:             "production",
		JWTSecret:               "a-production-secret-that-is-long-enough",
		CORSAllowedOrigins:      []string{"https://roomies.example"},
		Port:                    "8080",
		WorkerHealthPort:        "8081",
		PublicBaseURL:           "https://roomies.example",
		InvitationTTL:           168,
		JWTAccessTTLHours:       8,
		JobPollInterval:         2,
		JobLeaseSeconds:         30,
		SMTPPort:                587,
		SMTPHost:                "smtp.example.com",
		SMTPFrom:                "Roomies <sender@example.com>",
		InvitationDeliveryKeyID: "test-key",
		InvitationDeliveryKey:   testDeliveryKey(),
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("expected valid SMTP settings: %v", err)
	}
	config.SMTPUsername = "user"
	if err := config.Validate(); err == nil {
		t.Fatal("expected username without password to be rejected")
	}
}

func TestConfigRequiresDedicatedNotificationKeyForPush(t *testing.T) {
	config := &Config{Environment: "test", JWTSecret: "test-secret", CORSAllowedOrigins: []string{"http://localhost"}, Port: "8080", WorkerHealthPort: "8081", PublicBaseURL: "http://localhost", InvitationTTL: 168, JWTAccessTTLHours: 8, JobPollInterval: 2, JobLeaseSeconds: 30, WebPushPublicKey: "public", WebPushPrivateKey: "private"}
	if err := config.Validate(); err == nil {
		t.Fatal("expected push without notification encryption key to fail")
	}
	config.NotificationDeliveryKeyID = "notification-v1"
	config.NotificationDeliveryKey = testDeliveryKey()
	if err := config.Validate(); err != nil {
		t.Fatalf("expected dedicated key to validate: %v", err)
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
		JWTAccessTTLHours:  8,
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

func TestDevelopmentSecretOnlyAllowedOnLoopback(t *testing.T) {
	base := func(publicURL, secret string) *Config {
		return &Config{
			Environment:        "development",
			JWTSecret:          secret,
			CORSAllowedOrigins: []string{publicURL},
			Port:               "8080",
			WorkerHealthPort:   "8081",
			PublicBaseURL:      publicURL,
			InvitationTTL:      168,
			JWTAccessTTLHours:  8,
			JobPollInterval:    2,
			JobLeaseSeconds:    30,
		}
	}
	for _, local := range []string{"http://localhost:8080", "http://127.0.0.1:8787", "http://[::1]:8080"} {
		if err := base(local, developmentJWTSecret).Validate(); err != nil {
			t.Fatalf("%s: dev secret should be allowed locally: %v", local, err)
		}
	}
	for _, public := range []string{"https://roomies-dev.example.com", "http://192.168.1.20:8080", "not a url"} {
		if err := base(public, developmentJWTSecret).Validate(); err == nil {
			t.Fatalf("%s: dev secret must be rejected off loopback", public)
		}
	}
	if err := base("https://roomies-dev.example.com", "an-explicit-dev-secret").Validate(); err != nil {
		t.Fatalf("explicit secret should be accepted: %v", err)
	}
}
