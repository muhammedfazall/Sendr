package authhandler

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

type Handler struct {
	svc         ports.AuthService
	frontendURL string
}

func New(svc ports.AuthService, frontendURL string) *Handler {
	return &Handler{svc: svc, frontendURL: frontendURL}
}

// GET /auth/google
func (h *Handler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := uuid.NewString()
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth_state",
			Value:    state,
			MaxAge:   300,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})
		http.Redirect(w, r, h.svc.GetAuthURL(state), http.StatusTemporaryRedirect)
	}
}

// GET /auth/google/callback
// Redirects to frontend /callback?token=<jwt> on success
// Redirects to frontend /callback?error=auth_failed on failure
func (h *Handler) Callback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("oauth_state")
		if err != nil || cookie.Value != r.URL.Query().Get("state") {
			response.Error(w, http.StatusBadRequest, "invalid_state", "OAuth state mismatch")
			return
		}

		token, err := h.svc.HandleCallback(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			fmt.Println("CALLBACK ERROR:", err)
			http.Redirect(w, r,
				fmt.Sprintf("%s/callback?error=auth_failed", h.frontendURL),
				http.StatusTemporaryRedirect)
			return
		}

		http.Redirect(w, r,
			fmt.Sprintf("%s/callback?token=%s", h.frontendURL, token),
			http.StatusTemporaryRedirect)
	}
}