package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Raw env values; each consumer validates its own
type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	MailURL             string
	IsGuestRegistration bool
	FirebaseConfig      string
}

// Missing .env is fine; env vars always win over .env
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("config: loading .env: %w", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Required: server always applies migrations at startup
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL must be set")
	}

	return &Config{
		Port:                port,
		DatabaseURL:         databaseURL,
		JWTSecret:           os.Getenv("JWT_SECRET"),
		MailURL:             os.Getenv("MAIL_URL"),
		IsGuestRegistration: os.Getenv("IS_GUEST_REGISTRATION") == "1",
		FirebaseConfig:      os.Getenv("FIREBASE_CONFIG"),
	}, nil
}
