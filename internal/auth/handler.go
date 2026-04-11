package auth

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// Handler handles HTTP requests for authentication.
type Handler struct {
	service *Service
}

// NewHandler creates a new auth Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GoogleLogin redirects the user to Google's consent screen.
func (h *Handler) GoogleLogin() http.HandlerFunc {
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
		url := h.service.GetAuthURL(state)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

// GoogleCallback handles the redirect back from Google.
func (h *Handler) GoogleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1 — verify state matches what we set in the cookie
		cookie, err := r.Cookie("oauth_state")
		stateParam := r.URL.Query().Get("state")

		if err != nil || cookie.Value != stateParam {
			http.Error(w, "invalid state", http.StatusBadRequest)
			return
		}

		// Step 2 — delegate to service for token exchange, user upsert, JWT signing
		code := r.URL.Query().Get("code")
		tokens, err := h.service.HandleGoogleCallback(r.Context(), code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokens)
	}
}
