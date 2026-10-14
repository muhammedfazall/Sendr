package mehandler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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
func (h *Handler) Get(c *gin.Context) {
	userID, ok := userIDFromRequest(c)
	if !ok {
		return
	}

	user, plan, err := h.users.FindWithPlan(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "fetch_failed", "could not load profile")
		return
	}

	usageToday, remaining := h.usageSnapshot(c, userID, plan.DailyLimit)
	writeProfile(c, http.StatusOK, user, plan, usageToday, remaining)
}

// PATCH /me updates editable profile fields.
func (h *Handler) Update(c *gin.Context) {
	userID, ok := userIDFromRequest(c)
	if !ok {
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_body", "invalid profile payload")
		return
	}

	name := strings.TrimSpace(body.Name)
	nameLength := len([]rune(name))
	if nameLength < 2 || nameLength > 80 {
		response.Error(c, http.StatusBadRequest, "invalid_name", "name must be between 2 and 80 characters")
		return
	}

	if _, err := h.users.UpdateProfile(c.Request.Context(), userID, name); err != nil {
		response.Error(c, http.StatusInternalServerError, "update_failed", "could not update profile")
		return
	}

	user, plan, err := h.users.FindWithPlan(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "fetch_failed", "could not load profile")
		return
	}

	usageToday, remaining := h.usageSnapshot(c, userID, plan.DailyLimit)
	writeProfile(c, http.StatusOK, user, plan, usageToday, remaining)
}

func (h *Handler) usageSnapshot(c *gin.Context, userID string, dailyLimit int) (int, int) {
	usageToday := 0
	remaining := dailyLimit
	if dailyLimit < 0 {
		remaining = -1
	}

	count, err := h.limiter.GetCount(c.Request.Context(), userID)
	if err != nil {
		middleware.Logger(c).Error("failed to read usage count", "user_id", userID, "err", err)
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

func userIDFromRequest(c *gin.Context) (string, bool) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
		return "", false
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "invalid token claims")
		return "", false
	}
	return userID, true
}

func writeProfile(c *gin.Context, status int, user *domain.User, plan *domain.Plan, usageToday, remaining int) {
	response.JSON(c, status, map[string]any{
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
