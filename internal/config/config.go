package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServerPort  string
	ServerHost  string
	Environment string

	DatabaseURL string

	JWTSecret         string
	JWTExpiration     time.Duration
	RefreshExpiration time.Duration

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	InternalAPIKey string
	EncryptionKey  string

	AllowedOrigins []string
}

func Load() *Config {
	cfg := &Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		ServerHost:        getEnv("SERVER_HOST", "0.0.0.0"),
		Environment:       getEnv("ENVIRONMENT", "development"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://aipsa:aipsa_secret@localhost:5432/aipsa_platform?sslmode=disable"),
		JWTExpiration:     getDurationEnv("JWT_EXPIRATION", 15*time.Minute),
		RefreshExpiration: getDurationEnv("REFRESH_EXPIRATION", 168*time.Hour),
		GoogleClientID:    os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL: getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/v1/auth/google/callback"),
		AllowedOrigins:    getListEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080"),
	}

	cfg.JWTSecret = getEnv("JWT_SECRET", generateRandomKey(32))
	cfg.InternalAPIKey = getEnv("INTERNAL_API_KEY", generateRandomKey(32))
	cfg.EncryptionKey = getEnv("ENCRYPTION_KEY", generateRandomKey(32))

	return cfg
}

func (c *Config) IsGoogleOAuthEnabled() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getListEnv(key, defaultValue string) []string {
	value := getEnv(key, defaultValue)
	if value == "" {
		return []string{}
	}
	return strings.Split(value, ",")
}

func generateRandomKey(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString(make([]byte, length))
	}
	return hex.EncodeToString(b)
}
