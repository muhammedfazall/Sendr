package apikeyhandler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
func (h *Handler) Create(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		response.Error(c, http.StatusNotFound, "notfound", "not found")
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil || body.Name == "" {
		response.Error(c, http.StatusBadRequest, "invalid_body", "name is required")
		return
	}

	fullKey, key, err := h.svc.Create(c.Request.Context(), userID, body.Name)
	if err != nil {
		middleware.Logger(c).Error("create api key failed", "err", err)
		response.Error(c, http.StatusInternalServerError, "create_failed", "could not create API key")
		return
	}

	response.JSON(c, http.StatusCreated, map[string]any{
		"id":      key.ID,
		"name":    key.Name,
		"prefix":  key.Prefix,
		"api_key": fullKey, // shown once — never stored plain
	})
}

// GET /apikeys
func (h *Handler) List(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
		return
	}
	userID := claims["user_id"].(string)

	keys, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		middleware.Logger(c).Error("list api keys failed", "err", err)
		response.Error(c, http.StatusInternalServerError, "list_failed", "could not list API keys")
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
	response.JSON(c, http.StatusOK, out)
}

// DELETE /apikeys/:id
func (h *Handler) Revoke(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "invalid token claims")
		return
	}

	keyID := c.Param("id")
	if err := h.svc.Revoke(c.Request.Context(), keyID, userID); err != nil {
		response.Error(c, http.StatusNotFound, "not_found", "key not found or already revoked")
		return
	}

	c.Status(http.StatusNoContent)
}
