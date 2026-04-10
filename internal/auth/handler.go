package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// oauthConfig builds the Google OAuth config from your .env values
func oauthConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// GoogleLogin redirects the user to Google's consent screen
func GoogleLogin(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Generate a random state token to prevent CSRF attacks
		state := uuid.NewString()

		// Store it in a short-lived cookie — we'll verify it in the callback
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    state,
			MaxAge:   300, // expires in 5 minutes
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})

		// Redirect browser to Google
		url := oauthConfig(cfg).AuthCodeURL(state)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

// googleUser is what Google's userinfo API returns
type googleUser struct {
	ID    string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GoogleCallback handles the redirect back from Google
func GoogleCallback(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1 — verify state matches what we set in the cookie
		cookie, err := r.Cookie("oauth_state")

		stateParam := r.URL.Query().Get("state")

		// log.Printf("cookie state: %v, url state: %v, err: %v",
		// 	func() string {
		// 		if err == nil {
		// 			return cookie.Value
		// 		}
		// 		return "NO COOKIE"
		// 	}(),
		// 	stateParam, err)

		if err != nil || cookie.Value != stateParam {
			http.Error(w, "invalid state", http.StatusBadRequest)
			return
		}

		if err != nil || cookie.Value != r.URL.Query().Get("state") {
			http.Error(w, "invalid state", http.StatusBadRequest)
			return
		}

		// Step 2 — exchange the code Google gave us for an access token
		code := r.URL.Query().Get("code")
		token, err := oauthConfig(cfg).Exchange(context.Background(), code)
		if err != nil {
			http.Error(w, "failed to exchange token", http.StatusInternalServerError)
			return
		}

		// Step 3 — use the access token to fetch the user's profile from Google
		client := oauthConfig(cfg).Client(context.Background(), token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
		if err != nil {
			http.Error(w, "failed to get user info", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Step 4 — decode the profile JSON
		var gUser googleUser
		if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
			http.Error(w, "failed to decode user info", http.StatusInternalServerError)
			return
		}

		// Step 5 — sign a JWT and return it
		// (we'll add DB upsert here in a moment)
		jwtToken, err := signJWT(cfg, gUser.ID, gUser.Email)
		if err != nil {
			http.Error(w, "failed to sign token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": jwtToken})
	}
}

// signJWT creates a signed JWT token for the user
func signJWT(cfg *config.Config, userID, email string) (string, error) {
	// Load the private key from disk
	keyBytes, err := os.ReadFile(cfg.JWTPrivateKeyPath)
	if err != nil {
		return "", err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return "", err
	}

	// Build the claims — this is the data stored inside the token
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(), // expires in 24h
	}

	// Sign with RS256 (RSA + SHA256)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}
