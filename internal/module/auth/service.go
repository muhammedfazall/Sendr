package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Service contains the business logic for authentication.
type Service struct {
	repo *Repository
	cfg  *config.Config
}

// NewService creates a new auth Service.
func NewService(repo *Repository, cfg *config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

// OAuthConfig builds the Google OAuth2 config from environment values.
func (s *Service) OAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.GoogleClientID,
		ClientSecret: s.cfg.GoogleClientSecret,
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// GetAuthURL returns the Google OAuth consent screen URL for the given state.
func (s *Service) GetAuthURL(state string) string {
	return s.OAuthConfig().AuthCodeURL(state)
}

// HandleGoogleCallback exchanges the auth code, fetches the Google user profile,
// upserts the user in the database, and returns signed JWT tokens.
func (s *Service) HandleGoogleCallback(ctx context.Context, code string) (*AuthTokens, error) {
	// Exchange code for OAuth token
	oauthToken, err := s.OAuthConfig().Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	// Fetch user info from Google
	client := s.OAuthConfig().Client(ctx, oauthToken)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var gUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Upsert user in DB
	userID, err := s.repo.UpsertUser(ctx, gUser.ID, gUser.Email, gUser.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	// Sign JWT
	token, err := s.signJWT(userID, gUser.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &AuthTokens{Token: token}, nil
}

// signJWT creates a signed RS256 JWT token for the given user.
func (s *Service) signJWT(userID, email string) (string, error) {
	keyBytes, err := os.ReadFile(s.cfg.JWTPrivateKeyPath)
	if err != nil {
		return "", err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}
