package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	MailURL             string
	IsGuestRegistration bool
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

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("config: JWT_SECRET must be set to at least 32 bytes")
	}

	// Required: server always applies migrations at startup
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL must be set")
	}

	// Unset = no email; see auth.NoopLoginCodeSender
	mailURL := os.Getenv("MAIL_URL")

	// Unset/not 1: RequestLogin never creates an account
	isGuestRegistration := os.Getenv("IS_GUEST_REGISTRATION") == "1"

	return &Config{
		Port:                port,
		DatabaseURL:         databaseURL,
		JWTSecret:           jwtSecret,
		MailURL:             mailURL,
		IsGuestRegistration: isGuestRegistration,
	}, nil
}
