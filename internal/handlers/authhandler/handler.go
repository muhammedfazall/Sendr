package authhandler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

type Handler struct {
	svc ports.AuthService
}

func New(svc ports.AuthService) *Handler {
	return &Handler{svc: svc}
}

// GET /auth/google — redirect to Google consent screen
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

// GET /auth/google/callback — exchange code, upsert user, return JWT
func (h *Handler) Callback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("oauth_state")
		if err != nil || cookie.Value != r.URL.Query().Get("state") {
			response.Error(w, http.StatusBadRequest, "invalid_state", "OAuth state mismatch")
			return
		}

		token, err := h.svc.HandleCallback(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "auth_failed", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}