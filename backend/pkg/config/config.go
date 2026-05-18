package config

import (
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                string
	DBUrl                 string
	RedisUrl              string
	JWTPrivateKeyPEM      string
	JWTPublicKeyPEM       string
	JWTPrivateKeyPath     string
	JWTPublicKeyPath      string
	GoogleClientID        string
	GoogleClientSecret    string
	OAuthStateSecret      string
	SendGridKey           string
	FromEmail             string
	FromName              string
	FrontendURL           string
	Port                  string
	BackendURL            string
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string
	AllowedOrigins        []string
}

func Load() (*Config, error) {
	appEnv := getEnvOrDefault("APP_ENV", "development")
	if appEnv != "production" {
		_ = godotenv.Load()
	}

	cfg := &Config{
		AppEnv:                appEnv,
		DBUrl:                 os.Getenv("DB_URL"),
		RedisUrl:              os.Getenv("REDIS_URL"),
		JWTPrivateKeyPEM:      os.Getenv("JWT_PRIVATE_KEY_PEM"),
		JWTPublicKeyPEM:       os.Getenv("JWT_PUBLIC_KEY_PEM"),
		JWTPrivateKeyPath:     os.Getenv("JWT_PRIVATE_KEY_PATH"),
		JWTPublicKeyPath:      os.Getenv("JWT_PUBLIC_KEY_PATH"),
		GoogleClientID:        os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:    os.Getenv("GOOGLE_CLIENT_SECRET"),
		OAuthStateSecret:      os.Getenv("OAUTH_STATE_SECRET"),
		SendGridKey:           os.Getenv("SENDGRID_KEY"),
		RazorpayKeyID:         os.Getenv("RAZORPAY_KEY_ID"),
		RazorpayKeySecret:     os.Getenv("RAZORPAY_KEY_SECRET"),
		RazorpayWebhookSecret: os.Getenv("RAZORPAY_WEBHOOK_SECRET"),
		FromEmail:             getEnvOrDefault("FROM_EMAIL", "noreply@example.com"),
		FromName:              getEnvOrDefault("FROM_NAME", "Sendr"),
		FrontendURL:           getEnvOrDefault("FRONTEND_URL", "http://localhost:5173"),
		Port:                  getEnvOrDefault("PORT", "8080"),
		BackendURL:            getEnvOrDefault("BACKEND_URL", "http://localhost:8080"),
		AllowedOrigins:        splitCSV(getEnvOrDefault("ALLOWED_ORIGINS", getEnvOrDefault("FRONTEND_URL", "http://localhost:5173"))),
	}

	// Validate required fields
	required := map[string]string{
		"DB_URL":               cfg.DBUrl,
		"REDIS_URL":            cfg.RedisUrl,
		"GOOGLE_CLIENT_ID":     cfg.GoogleClientID,
		"GOOGLE_CLIENT_SECRET": cfg.GoogleClientSecret,
		"OAUTH_STATE_SECRET":   cfg.OAuthStateSecret,
		"SENDGRID_KEY":         cfg.SendGridKey,
		"RAZORPAY_KEY_ID":      cfg.RazorpayKeyID,
		"RAZORPAY_KEY_SECRET":  cfg.RazorpayKeySecret,
	}

	var missing []string
	for k, v := range required {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if cfg.JWTPrivateKeyPEM == "" && cfg.JWTPrivateKeyPath == "" {
		missing = append(missing, "JWT_PRIVATE_KEY_PEM or JWT_PRIVATE_KEY_PATH")
	}
	if cfg.JWTPublicKeyPEM == "" && cfg.JWTPublicKeyPath == "" {
		missing = append(missing, "JWT_PUBLIC_KEY_PEM or JWT_PUBLIC_KEY_PATH")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %v", missing)
	}

	for _, origin := range cfg.AllowedOrigins {
		if err := validateOrigin(origin); err != nil {
			return nil, err
		}
	}
	if err := validateOrigin(cfg.FrontendURL); err != nil {
		return nil, fmt.Errorf("invalid FRONTEND_URL: %w", err)
	}
	if err := validateOrigin(cfg.BackendURL); err != nil {
		return nil, fmt.Errorf("invalid BACKEND_URL: %w", err)
	}
	if _, err := mail.ParseAddress(cfg.FromEmail); err != nil {
		return nil, fmt.Errorf("invalid FROM_EMAIL: %w", err)
	}

	return cfg, nil
}

func (c *Config) JWTPrivateKeyBytes() ([]byte, error) {
	return keyBytes(c.JWTPrivateKeyPEM, c.JWTPrivateKeyPath, "JWT_PRIVATE_KEY")
}

func (c *Config) JWTPublicKeyBytes() ([]byte, error) {
	return keyBytes(c.JWTPublicKeyPEM, c.JWTPublicKeyPath, "JWT_PUBLIC_KEY")
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}

	return out
}

func keyBytes(pemValue, path, name string) ([]byte, error) {
	if pemValue != "" {
		return []byte(normalizePEM(pemValue)), nil
	}
	if path == "" {
		return nil, fmt.Errorf("%s_PEM or %s_PATH is required", name, name)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s_PATH: %w", name, err)
	}
	return raw, nil
}

func normalizePEM(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), `\n`, "\n")
}

func validateOrigin(origin string) error {
	if origin == "" || origin == "*" {
		return fmt.Errorf("invalid allowed origin: %q", origin)
	}

	u, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("invalid allowed origin %q: %w", origin, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("allowed origin %q must use http or https", origin)
	}

	if u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("allowed origin %q must be an origin only, like https://sendr.app", origin)
	}

	return nil
}
