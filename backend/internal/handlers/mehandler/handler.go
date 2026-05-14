package mehandler

import (
	"log"
	"net/http"

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

// GET /me — returns authenticated user's profile + plan
func (h *Handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token claims")
			return
		}

		user, plan, err := h.users.FindWithPlan(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "fetch_failed", "could not load profile")
			return
		}

		usageToday := 0
		remaining := plan.DailyLimit

		count, err := h.limiter.GetCount(r.Context(), userID)
		if err != nil {
			// Non-fatal: log and fall back to 0 usage
			log.Printf("mehandler: failed to read usage count for %s: %v", userID, err)
		} else {
			usageToday = count
			remaining = plan.DailyLimit - count
			if remaining < 0 {
				remaining = 0
			}
		}

		response.JSON(w, http.StatusOK, map[string]any{
			"id":          user.ID,
			"email":       user.Email,
			"name":        user.Name,
			"plan":        plan.Name,
			"daily_limit": plan.DailyLimit,
			"max_api_keys":   plan.MaxAPIKeys,
			"rate_wait_secs": plan.RateWaitSecs,
			"usage_today": usageToday,
			"remaining":   remaining,
		})
	}
}
