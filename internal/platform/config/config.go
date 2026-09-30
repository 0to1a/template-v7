package config

import (
	"encoding/base64"
	"encoding/json"
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
	Firebase            *FirebaseConfig
}

// Firebase web config JSON (camelCase keys, as the console shows it)
type FirebaseConfig struct {
	APIKey     string `json:"apiKey"`
	AuthDomain string `json:"authDomain"`
	ProjectID  string `json:"projectId"`
	AppID      string `json:"appId"`
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

	// Unset = Google login disabled; malformed aborts startup
	firebase, err := parseFirebaseConfig(os.Getenv("FIREBASE_CONFIG"))
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:                port,
		DatabaseURL:         databaseURL,
		JWTSecret:           jwtSecret,
		MailURL:             mailURL,
		IsGuestRegistration: isGuestRegistration,
		Firebase:            firebase,
	}, nil
}

func parseFirebaseConfig(encoded string) (*FirebaseConfig, error) {
	if encoded == "" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("config: FIREBASE_CONFIG must be base64: %w", err)
	}
	var cfg FirebaseConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("config: FIREBASE_CONFIG must be JSON: %w", err)
	}
	if cfg.APIKey == "" || cfg.AuthDomain == "" || cfg.ProjectID == "" || cfg.AppID == "" {
		return nil, fmt.Errorf("config: FIREBASE_CONFIG needs apiKey, authDomain, projectId and appId")
	}
	return &cfg, nil
}
