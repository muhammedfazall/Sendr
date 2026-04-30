package services

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type authService struct {
	users      ports.UserRepository
	tokens     ports.TokenStore
	cfg        *config.Config
	privateKey *rsa.PrivateKey
}

// NewAuthService wires up the auth service with its dependencies.
func NewAuthService(users ports.UserRepository, tokens ports.TokenStore, cfg *config.Config) ports.AuthService {
	keyBytes, err := os.ReadFile(cfg.JWTPrivateKeyPath)
	if err != nil {
		log.Fatalf("read private key: %v", err)
	}
	pk, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		log.Fatalf("parse private key: %v", err)
	}
	return &authService{users: users, tokens: tokens, cfg: cfg, privateKey: pk}
}

func (s *authService) oauthCfg() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.GoogleClientID,
		ClientSecret: s.cfg.GoogleClientSecret,
		RedirectURL:  fmt.Sprintf("%s/auth/google/callback", s.cfg.BackendURL),
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func (s *authService) GetAuthURL(state string) string {
	return s.oauthCfg().AuthCodeURL(state)
}

// HandleCallback exchanges the OAuth code, upserts the user, then returns
// an access token (15 min) and a refresh token (7 days).
func (s *authService) HandleCallback(ctx context.Context, code string) (string, string, error) {
	tok, err := s.oauthCfg().Exchange(ctx, code)
	if err != nil {
		return "", "", fmt.Errorf("token exchange: %w", err)
	}

	resp, err := s.oauthCfg().Client(ctx, tok).Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return "", "", fmt.Errorf("userinfo fetch: %w", err)
	}
	defer resp.Body.Close()

	var g struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
		return "", "", fmt.Errorf("userinfo decode: %w", err)
	}

	user, err := s.users.Upsert(ctx, g.Sub, g.Email, g.Name)
	if err != nil {
		return "", "", fmt.Errorf("upsert user: %w", err)
	}

	return s.issueTokenPair(ctx, user.ID, user.Email)
}

// RefreshToken validates the existing refresh token, rotates it, and issues
// a new access + refresh pair. The old refresh token is invalidated.
func (s *authService) RefreshToken(ctx context.Context, userID, refreshTokenID string) (string, string, error) {
	valid, err := s.tokens.Validate(ctx, userID, refreshTokenID)
	if err != nil {
		return "", "", fmt.Errorf("validate refresh token: %w", err)
	}
	if !valid {
		return "", "", fmt.Errorf("invalid or expired refresh token")
	}

	// Look up user to get email for the JWT claims
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("find user: %w", err)
	}

	return s.issueTokenPair(ctx, user.ID, user.Email)
}

// Logout deletes the refresh token from Redis.
func (s *authService) Logout(ctx context.Context, userID string) error {
	return s.tokens.Delete(ctx, userID)
}

// issueTokenPair creates a new access JWT + refresh token and stores the
// refresh token in Redis. Returns (accessToken, refreshToken, error).
func (s *authService) issueTokenPair(ctx context.Context, userID, email string) (string, string, error) {
	accessToken, err := s.signJWT(userID, email)
	if err != nil {
		return "", "", fmt.Errorf("sign JWT: %w", err)
	}

	refreshID := uuid.NewString()
	if err := s.tokens.Store(ctx, userID, refreshID, constants.RefreshTokenExpiry); err != nil {
		return "", "", fmt.Errorf("store refresh token: %w", err)
	}

	return accessToken, refreshID, nil
}

func (s *authService) signJWT(userID, email string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"iss":     "sendr",
		"aud":     "sendr-api",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(constants.JWTExpiry).Unix(),
	}).SignedString(s.privateKey)
}
