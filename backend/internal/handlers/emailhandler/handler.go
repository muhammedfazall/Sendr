package emailhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

// statsProvider returns aggregate delivery stats for a user.
type statsProvider interface {
	GetStats(ctx context.Context, userID string) (*ports.EmailEventStats, error)
}

// jobReader defines the job read methods the handler needs.
type jobReader interface {
	GetByID(ctx context.Context, jobID string) (*domain.Job, error)
	ListByUser(ctx context.Context, userID, status string, limit, offset int) ([]domain.Job, error)
}

const (
	maxSubjectLength = 998
	maxBodyLength    = 50_000
)

// Handler handles POST /emails/send, GET /emails/:id, and GET /emails/stats.
// Routes under this handler are protected by the API key middleware,
// not JWT — callers authenticate with their mk_live_... key.
type Handler struct {
	email     ports.EmailService
	jobDB     jobReader
	stats     statsProvider
	templates ports.TemplateService
}

func New(email ports.EmailService, jobDB jobReader, stats statsProvider, templates ports.TemplateService) *Handler {
	return &Handler{email: email, jobDB: jobDB, stats: stats, templates: templates}
}

// sendRequest is the JSON body for POST /emails/send with optional template support.
type sendRequest struct {
	domain.EmailPayload
	TemplateID   string         `json:"template_id,omitempty"`
	TemplateData map[string]any `json:"template_data,omitempty"`
}

// Send handles POST /emails/send.
// Expects Authorization: Bearer mk_live_<prefix>.<secret>
// Returns 202 Accepted with the job ID.
func (h *Handler) Send(c *gin.Context) {
	// The API key middleware already validated the key and injected it into context.
	// We pull the raw key back out to pass to EmailService (which re-validates
	// for rate limiting and queuing — single responsibility stays clean).
	authHeader := c.GetHeader("Authorization")
	fullKey := ""
	if len(authHeader) > 7 {
		fullKey = authHeader[7:] // strip "Bearer "
	}

	var req sendRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_body", "request body is not valid JSON")
		return
	}

	if req.TemplateID != "" {
		userID := userFromClaims(c)
		if userID == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized", "could not resolve user")
			return
		}
		subject, htmlBody, textBody, err := h.templates.Render(c.Request.Context(), req.TemplateID, userID, req.TemplateData)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "template_error", err.Error())
			return
		}
		if req.Subject == "" {
			req.Subject = subject
		}
		if req.HTMLBody == "" {
			req.HTMLBody = htmlBody
		}
		if req.TextBody == "" {
			req.TextBody = textBody
		}
	}

	if len(req.To) == 0 {
		response.Error(c, http.StatusBadRequest, "missing_fields", "to is required")
		return
	}
	for _, addr := range req.To {
		if _, err := mail.ParseAddress(addr); err != nil {
			response.Error(c, http.StatusBadRequest, "invalid_email", fmt.Sprintf("invalid recipient: %s", addr))
			return
		}
	}
	req.Subject = strings.TrimSpace(req.Subject)
	if req.Subject == "" {
		response.Error(c, http.StatusBadRequest, "missing_fields", "subject is required")
		return
	}
	if len(req.Subject) > maxSubjectLength {
		response.Error(c, http.StatusBadRequest, "subject_too_large", "subject must be under 998 characters")
		return
	}
	if req.TextBody == "" && req.HTMLBody == "" {
		response.Error(c, http.StatusBadRequest, "missing_fields", "text_body or html_body is required")
		return
	}
	if len(req.TextBody) > maxBodyLength {
		response.Error(c, http.StatusBadRequest, "body_too_large", "text_body must be under 50KB")
		return
	}
	if len(req.HTMLBody) > maxBodyLength {
		response.Error(c, http.StatusBadRequest, "body_too_large", "html_body must be under 50KB")
		return
	}

	job, err := h.email.Send(c.Request.Context(), fullKey, req.EmailPayload)
	if err != nil {
		h.handleSendError(c, err)
		return
	}

	response.JSON(c, http.StatusAccepted, map[string]string{
		"job_id":  job.ID,
		"message": "email queued",
	})
}

// GetJob handles GET /emails/:id — status polling for async sends.
func (h *Handler) GetJob(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		response.Error(c, http.StatusBadRequest, "missing_id", "job id is required")
		return
	}

	job, err := h.jobDB.GetByID(c.Request.Context(), jobID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "not_found", "job not found")
		return
	}

	response.JSON(c, http.StatusOK, map[string]any{
		"id":         job.ID,
		"status":     job.Status,
		"retries":    job.Retries,
		"created_at": job.CreatedAt,
		"updated_at": job.UpdatedAt,
	})
}

// Stats handles GET /emails/stats — returns delivery stats for the authenticated user.
func (h *Handler) Stats(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "invalid token claims")
		return
	}

	stats, err := h.stats.GetStats(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "failed to fetch stats")
		return
	}

	response.JSON(c, http.StatusOK, stats)
}

// handleSendError maps service-layer sentinel errors to HTTP responses.
// Also writes rate limit headers on 429 so callers know when to retry.
func (h *Handler) handleSendError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, constants.ErrRateLimitExceeded):
		// Tell the client when the window resets (next UTC midnight)
		reset := nextMidnightUTC()
		c.Header("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		c.Header("Retry-After", strconv.FormatInt(reset-time.Now().Unix(), 10))
		response.Error(c, http.StatusTooManyRequests, "rate_limit_exceeded", "daily email limit reached")

	case errors.Is(err, constants.ErrAPIKeyInvalid),
		errors.Is(err, constants.ErrAPIKeyRevoked),
		errors.Is(err, constants.ErrAPIKeyNotFound):
		response.Error(c, http.StatusUnauthorized, "invalid_api_key", "API key is invalid or revoked")

	case errors.Is(err, constants.ErrUserNotFound):
		response.Error(c, http.StatusUnauthorized, "user_not_found", "user associated with this key was not found")

	default:
		response.Error(c, http.StatusInternalServerError, "internal_error", "failed to queue email")
	}
}

func nextMidnightUTC() int64 {
	now := time.Now().UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return midnight.Unix()
}

func userFromClaims(c *gin.Context) string {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		return ""
	}
	userID, _ := claims["user_id"].(string)
	return userID
}

// List handles GET /emails — returns the authenticated user's email history.
// Protected by JWT. Supports ?status=sent|pending|failed&limit=20&offset=0
func (h *Handler) List(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing claims")
		return
	}
	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "invalid token claims")
		return
	}

	// Parse query params with safe defaults
	validStatuses := map[string]bool{"pending": true, "processing": true, "sent": true, "failed": true}
	status := c.Query("status")
	if status != "" && !validStatuses[status] {
		response.Error(c, http.StatusBadRequest, "invalid_status", "status must be one of: pending, processing, sent, failed")
		return
	}

	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	jobs, err := h.jobDB.ListByUser(c.Request.Context(), userID, status, limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", "failed to fetch emails")
		return
	}

	response.JSON(c, http.StatusOK, jobs)
}
