package emailhandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/muhammedfazall/Sendr/internal/adapters/jobrepo"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

// Handler handles POST /emails/send and GET /emails/:id.
// Routes under this handler are protected by the API key middleware,
// not JWT — callers authenticate with their mk_live_... key.
type Handler struct {
	email  ports.EmailService
	jobDB  *jobrepo.PostgresJobRepository
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

		job, err := h.email.Send(r.Context(), fullKey, payload)
		if err != nil {
			h.handleSendError(w, err)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		response.JSON(w, http.StatusAccepted, map[string]string{
			"job_id":  job.ID,
			"message": "email queued",
		})
	}
}

// GetJob handles GET /emails/{id} — status polling for async sends.
func (h *Handler) GetJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("id")
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