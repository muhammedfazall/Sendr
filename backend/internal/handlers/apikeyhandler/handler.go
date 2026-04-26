package apikeyhandler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

type Handler struct {
	svc ports.APIKeyService
}

func New(svc ports.APIKeyService) *Handler {
	return &Handler{svc: svc}
}

// POST /apikeys
func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		userID, ok := claims["user_id"].(string)
		if !ok {
			response.Error(w, http.StatusNotFound, "notfound", "not found")
			return
		}

		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			response.Error(w, http.StatusBadRequest, "invalid_body", "name is required")
			return
		}

		fullKey, key, err := h.svc.Create(r.Context(), userID, body.Name)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "create_failed", err.Error())
			return
		}

		response.JSON(w, http.StatusCreated, map[string]any{
			"id":      key.ID,
			"name":    key.Name,
			"prefix":  key.Prefix,
			"api_key": fullKey, // shown once — never stored plain
		})
	}
}

// GET /apikeys
func (h *Handler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		userID := claims["user_id"].(string)

		keys, err := h.svc.List(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "list_failed", err.Error())
			return
		}

		// Never return hashed_key — map to safe shape
		type safeKey struct {
			ID        string    `json:"id"`
			Name      string    `json:"name"`
			Prefix    string    `json:"prefix"`
			CreatedAt time.Time `json:"created_at"`
		}
		out := make([]safeKey, len(keys))
		for i, k := range keys {
			out[i] = safeKey{ID: k.ID, Name: k.Name, Prefix: k.Prefix, CreatedAt: k.CreatedAt}
		}
		response.JSON(w, http.StatusOK, out)
	}
}

// DELETE /apikeys/{id}
func (h *Handler) Revoke() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		userID := claims["user_id"].(string)
		keyID := r.PathValue("id")

		if err := h.svc.Revoke(r.Context(), keyID, userID); err != nil {
			response.Error(w, http.StatusNotFound, "not_found", "key not found or already revoked")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}