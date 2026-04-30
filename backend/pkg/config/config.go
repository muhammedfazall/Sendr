package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl              string
	RedisUrl           string
	JWTPrivateKeyPath  string
	JWTPublicKeyPath   string
	GoogleClientID     string
	GoogleClientSecret string
	SendGridKey        string
	FromEmail          string
	FromName           string
	FrontendURL        string
	Port               string
	BackendURL         string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DBUrl:              os.Getenv("DB_URL"),
		RedisUrl:           os.Getenv("REDIS_URL"),
		JWTPrivateKeyPath:  os.Getenv("JWT_PRIVATE_KEY_PATH"),
		JWTPublicKeyPath:   os.Getenv("JWT_PUBLIC_KEY_PATH"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		SendGridKey:        os.Getenv("SENDGRID_KEY"),
		FromEmail:          getEnvOrDefault("FROM_EMAIL", "noreply@example.com"),
		FromName:           getEnvOrDefault("FROM_NAME", "Sendr"),
		FrontendURL:        getEnvOrDefault("FRONTEND_URL", "http://localhost:5173"),
		Port:               getEnvOrDefault("PORT", "8080"),
	}

	// Validate required fields
	required := map[string]string{
		"DB_URL":               cfg.DBUrl,
		"REDIS_URL":            cfg.RedisUrl,
		"JWT_PRIVATE_KEY_PATH": cfg.JWTPrivateKeyPath,
		"JWT_PUBLIC_KEY_PATH":  cfg.JWTPublicKeyPath,
		"GOOGLE_CLIENT_ID":     cfg.GoogleClientID,
		"GOOGLE_CLIENT_SECRET": cfg.GoogleClientSecret,
		"SENDGRID_KEY":         cfg.SendGridKey,
	}
	var missing []string
	for k, v := range required {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %v", missing)
	}
	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
