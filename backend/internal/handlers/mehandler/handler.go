package mehandler

import (
	"net/http"

	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

type Handler struct {
	users ports.UserRepository
}

func New(users ports.UserRepository) *Handler {
	return &Handler{users: users}
}

// GET /me — returns authenticated user's profile + plan
func (h *Handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		userID := claims["user_id"].(string)

		user, plan, err := h.users.FindWithPlan(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "fetch_failed", "could not load profile")
			return
		}

		response.JSON(w, http.StatusOK, map[string]any{
			"id":          user.ID,
			"email":       user.Email,
			"name":        user.Name,
			"plan":        plan.Name,
			"daily_limit": plan.DailyLimit,
		})
	}
}