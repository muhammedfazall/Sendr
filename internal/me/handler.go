package me

import (
	"encoding/json"
	"net/http"

	"github.com/muhammedfazall/Sendr/internal/middleware"
)

// Handler handles HTTP requests for the /me endpoint.
type Handler struct {
	service *Service
}

// NewHandler creates a new me Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Get handles GET /me — returns the authenticated user's profile.
func (h *Handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, _ := middleware.GetClaims(r)
		profile := h.service.GetProfile(claims)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}
}
