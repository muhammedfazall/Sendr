package apikey

import (
	"encoding/json"
	"net/http"

	"github.com/muhammedfazall/Sendr/internal/middleware"
)

// Handler handles HTTP requests for API key management.
type Handler struct {
	service *Service
}

// NewHandler creates a new apikey Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /apikeys — create a new API key.
func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID := claims["user_id"].(string)

		var body CreateKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		resp, err := h.service.CreateKey(r.Context(), userID, body.Name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

// List handles GET /apikeys — list all keys for the user.
func (h *Handler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID := claims["user_id"].(string)

		keys, err := h.service.ListKeys(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to fetch keys", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keys)
	}
}

// Revoke handles DELETE /apikeys/{id} — revoke a key.
func (h *Handler) Revoke() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID := claims["user_id"].(string)

		keyID := r.PathValue("id")

		if err := h.service.RevokeKey(r.Context(), keyID, userID); err != nil {
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
