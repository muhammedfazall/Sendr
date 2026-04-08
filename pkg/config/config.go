package config

import (
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
	SendGridKey         string
	Port               string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
        DBUrl:              os.Getenv("DB_URL"),
        RedisUrl:           os.Getenv("REDIS_URL"),
        JWTPrivateKeyPath:  os.Getenv("JWT_PRIVATE_KEY_PATH"),
        JWTPublicKeyPath:   os.Getenv("JWT_PUBLIC_KEY_PATH"),
        GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
        GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
        SendGridKey:        os.Getenv("SENDGRID_KEY"),
        Port:               getEnvOrDefault("PORT", "8080"),
    }, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key) ; v != "" {
		return v
	}
	return fallback
}