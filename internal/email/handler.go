package email

import (
	"encoding/json"
	"net/http"

	"github.com/muhammedfazall/Sendr/internal/job"
	"github.com/muhammedfazall/Sendr/internal/middleware"
)

type Handler struct {
	service *job.Service
}

func NewHandler(service *job.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Send() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		userID := claims["user_id"].(string)

		var body email.EmailPayload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		j, err := h.service.Enqueue(r.Context(), userID, "api-key-id-placeholder", body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"job_id": j.ID.String(),
		})
	}
}
