package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type authService struct {
	users ports.UserRepository
	cfg   *config.Config
}

// NewAuthService wires up the auth service with its dependencies.
func NewAuthService(users ports.UserRepository, cfg *config.Config) ports.AuthService {
	return &authService{users: users, cfg: cfg}
}

func (s *authService) oauthCfg() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.GoogleClientID,
		ClientSecret: s.cfg.GoogleClientSecret,
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func (s *authService) GetAuthURL(state string) string {
	return s.oauthCfg().AuthCodeURL(state)
}

func (s *authService) HandleCallback(ctx context.Context, code string) (string, error) {
    tok, err := s.oauthCfg().Exchange(ctx, code)
    if err != nil {
        fmt.Println("EXCHANGE ERROR:", err)
        return "", fmt.Errorf("userinfo fetch: %w", err)
    }
    fmt.Println("EXCHANGE OK")

    resp, err := s.oauthCfg().Client(ctx, tok).Get("https://www.googleapis.com/oauth2/v3/userinfo")
    if err != nil {
        fmt.Println("USERINFO ERROR:", err)
        return "", fmt.Errorf("userinfo fetch: %w", err)
    }
    fmt.Println("USERINFO OK")

    var g struct {
        Sub   string `json:"sub"`
        Email string `json:"email"`
        Name  string `json:"name"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
        fmt.Println("DECODE ERROR:", err)
        return "", fmt.Errorf("userinfo decode: %w", err)
    }
    fmt.Println("DECODED:", g.Email)

    user, err := s.users.Upsert(ctx, g.Sub, g.Email, g.Name)
    if err != nil {
        fmt.Println("UPSERT ERROR:", err)
        return "", fmt.Errorf("upsert user: %w", err)
    }
    fmt.Println("UPSERT OK:", user.ID)

    return s.signJWT(user.ID, user.Email)
}

func (s *authService) signJWT(userID, email string) (string, error) {
    keyBytes, err := os.ReadFile(s.cfg.JWTPrivateKeyPath)
    if err != nil {
        fmt.Println("READ KEY ERROR:", err)
        return "", fmt.Errorf("read private key: %w", err)
    }
    fmt.Println("KEY READ OK, length:", len(keyBytes))

    pk, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
    if err != nil {
        fmt.Println("PARSE KEY ERROR:", err)
        return "", fmt.Errorf("parse private key: %w", err)
    }
    fmt.Println("KEY PARSED OK")

    return jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
        "user_id": userID,
        "email":   email,
        "exp":     time.Now().Add(constants.JWTExpiry).Unix(),
    }).SignedString(pk)
}