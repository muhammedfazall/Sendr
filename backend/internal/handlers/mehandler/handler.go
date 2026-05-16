package mehandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

type Handler struct {
	users   ports.UserRepository
	limiter ports.RateLimiter
}

func New(users ports.UserRepository, limiter ports.RateLimiter) *Handler {
	return &Handler{users: users, limiter: limiter}
}

// GET /me returns authenticated user's profile and plan.
func (h *Handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromRequest(w, r)
		if !ok {
			return
		}

		user, plan, err := h.users.FindWithPlan(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "fetch_failed", "could not load profile")
			return
		}

		usageToday, remaining := h.usageSnapshot(r, userID, plan.DailyLimit)
		writeProfile(w, http.StatusOK, user, plan, usageToday, remaining)
	}
}

// PATCH /me updates editable profile fields.
func (h *Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromRequest(w, r)
		if !ok {
			return
		}

		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid_body", "invalid profile payload")
			return
		}

		name := strings.TrimSpace(body.Name)
		nameLength := len([]rune(name))
		if nameLength < 2 || nameLength > 80 {
			response.Error(w, http.StatusBadRequest, "invalid_name", "name must be between 2 and 80 characters")
			return
		}

		if _, err := h.users.UpdateProfile(r.Context(), userID, name); err != nil {
			response.Error(w, http.StatusInternalServerError, "update_failed", "could not update profile")
			return
		}

		user, plan, err := h.users.FindWithPlan(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "fetch_failed", "could not load profile")
			return
		}

		usageToday, remaining := h.usageSnapshot(r, userID, plan.DailyLimit)
		writeProfile(w, http.StatusOK, user, plan, usageToday, remaining)
	}
}

func (h *Handler) usageSnapshot(r *http.Request, userID string, dailyLimit int) (int, int) {
	usageToday := 0
	remaining := dailyLimit
	if dailyLimit < 0 {
		remaining = -1
	}

	count, err := h.limiter.GetCount(r.Context(), userID)
	if err != nil {
		log.Printf("mehandler: failed to read usage count for %s: %v", userID, err)
		return usageToday, remaining
	}

	usageToday = count
	if dailyLimit < 0 {
		return usageToday, -1
	}
	if usageToday > dailyLimit {
		usageToday = dailyLimit
	}

	remaining = dailyLimit - count
	if remaining < 0 {
		remaining = 0
	}
	return usageToday, remaining
}

func userIDFromRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := middleware.GetClaims(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
		return "", false
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token claims")
		return "", false
	}
	return userID, true
}

func writeProfile(w http.ResponseWriter, status int, user *domain.User, plan *domain.Plan, usageToday, remaining int) {
	response.JSON(w, status, map[string]any{
		"id":             user.ID,
		"email":          user.Email,
		"name":           user.Name,
		"created_at":     user.CreatedAt,
		"plan":           plan.Name,
		"daily_limit":    plan.DailyLimit,
		"max_api_keys":   plan.MaxAPIKeys,
		"rate_wait_secs": plan.RateWaitSecs,
		"usage_today":    usageToday,
		"remaining":      remaining,
	})
}
