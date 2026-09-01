package config

import "testing"

func TestProductionConfigRejectsWeakSecretAndWildcardCORS(t *testing.T) {
	config := &Config{Environment: "production", JWTSecret: developmentJWTSecret, CORSAllowedOrigins: []string{"https://roomies.example"}}
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
