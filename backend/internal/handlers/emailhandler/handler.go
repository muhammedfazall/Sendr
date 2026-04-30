package emailhandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/muhammedfazall/Sendr/internal/adapters/jobrepo"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

// Handler handles POST /emails/send and GET /emails/:id.
// Routes under this handler are protected by the API key middleware,
// not JWT — callers authenticate with their mk_live_... key.
type Handler struct {
	email ports.EmailService
	jobDB *jobrepo.PostgresJobRepository
}

func New(email ports.EmailService, jobDB *jobrepo.PostgresJobRepository) *Handler {
	return &Handler{email: email, jobDB: jobDB}
}

// Send handles POST /emails/send.
// Expects Authorization: Bearer mk_live_<prefix>.<secret>
// Returns 202 Accepted with the job ID.
func (h *Handler) Send() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The API key middleware already validated the key and injected it into context.
		// We pull the raw key back out to pass to EmailService (which re-validates
		// for rate limiting and queuing — single responsibility stays clean).
		authHeader := r.Header.Get("Authorization")
		fullKey := ""
		if len(authHeader) > 7 {
			fullKey = authHeader[7:] // strip "Bearer "
		}

		var payload domain.EmailPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
			return
		}
		if payload.To == "" || payload.Subject == "" || payload.Body == "" {
			response.Error(w, http.StatusBadRequest, "missing_fields", "to, subject and body are required")
			return
		}
		if _, err := mail.ParseAddress(payload.To); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid_email", "to is not a valid email address")
			return
		}
		if len(payload.Body) > 50_000 {
			response.Error(w, http.StatusBadRequest, "body_too_large", "body must be under 50KB")
			return
		}

		job, err := h.email.Send(r.Context(), fullKey, payload)
		if err != nil {
			h.handleSendError(w, err)
			return
		}
		response.JSON(w, http.StatusAccepted, map[string]string{
			"job_id":  job.ID,
			"message": "email queued",
		})
	}
}

// GetJob handles GET /emails/{id} — status polling for async sends.
func (h *Handler) GetJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "id")
		if jobID == "" {
			response.Error(w, http.StatusBadRequest, "missing_id", "job id is required")
			return
		}

		job, err := h.jobDB.GetByID(r.Context(), jobID)
		if err != nil {
			response.Error(w, http.StatusNotFound, "not_found", "job not found")
			return
		}

		response.JSON(w, http.StatusOK, map[string]any{
			"id":         job.ID,
			"status":     job.Status,
			"retries":    job.Retries,
			"created_at": job.CreatedAt,
			"updated_at": job.UpdatedAt,
		})
	}
}

// handleSendError maps service-layer sentinel errors to HTTP responses.
// Also writes rate limit headers on 429 so callers know when to retry.
func (h *Handler) handleSendError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, constants.ErrRateLimitExceeded):
		// Tell the client when the window resets (next UTC midnight)
		reset := nextMidnightUTC()
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		w.Header().Set("Retry-After", strconv.FormatInt(reset-time.Now().Unix(), 10))
		response.Error(w, http.StatusTooManyRequests, "rate_limit_exceeded", "daily email limit reached")

	case errors.Is(err, constants.ErrAPIKeyInvalid),
		errors.Is(err, constants.ErrAPIKeyRevoked),
		errors.Is(err, constants.ErrAPIKeyNotFound):
		response.Error(w, http.StatusUnauthorized, "invalid_api_key", "API key is invalid or revoked")

	case errors.Is(err, constants.ErrUserNotFound):
		response.Error(w, http.StatusUnauthorized, "user_not_found", "user associated with this key was not found")

	default:
		response.Error(w, http.StatusInternalServerError, "internal_error", "failed to queue email")
	}
}

func nextMidnightUTC() int64 {
	now := time.Now().UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return midnight.Unix()
}

// List handles GET /emails — returns the authenticated user's email history.
// Protected by JWT. Supports ?status=sent|pending|failed&limit=20&offset=0
func (h *Handler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing claims")
			return
		}
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token claims")
			return
		}

		// Parse query params with safe defaults
		validStatuses := map[string]bool{"pending": true, "processing": true, "sent": true, "failed": true}
		status := r.URL.Query().Get("status")
		if status != "" && !validStatuses[status] {
			response.Error(w, http.StatusBadRequest, "invalid_status", "status must be one of: pending, processing, sent, failed")
			return
		}

		limit := 20
		offset := 0

		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
				limit = v
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if v, err := strconv.Atoi(o); err == nil && v >= 0 {
				offset = v
			}
		}

		jobs, err := h.jobDB.ListByUser(r.Context(), userID, status, limit, offset)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "internal_error", "failed to fetch emails")
			return
		}

		response.JSON(w, http.StatusOK, jobs)
	}
}
