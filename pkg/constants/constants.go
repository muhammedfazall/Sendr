package constants

import (
	"errors"
	"net/http"
	"time"
)

// ── Durations ────────────────────────────────────────────────────────────────

const (
	JWTExpiry          = 24 * time.Hour
	OAuthStateCookieTTL = 5 * time.Minute
)

// ── HTTP status aliases ───────────────────────────────────────────────────────

const (
	StatusOK                  = http.StatusOK
	StatusCreated             = http.StatusCreated
	StatusAccepted            = http.StatusAccepted
	StatusBadRequest          = http.StatusBadRequest
	StatusUnauthorized        = http.StatusUnauthorized
	StatusForbidden           = http.StatusForbidden
	StatusNotFound            = http.StatusNotFound
	StatusConflict            = http.StatusConflict
	StatusTooManyRequests     = http.StatusTooManyRequests
	StatusInternalServerError = http.StatusInternalServerError
	StatusServiceUnavailable  = http.StatusServiceUnavailable
)

// ── Sentinel errors ───────────────────────────────────────────────────────────
// Use errors.Is() to match these in handlers.

var (
	// API key errors
	ErrAPIKeyInvalid  = errors.New("api key is invalid")
	ErrAPIKeyRevoked  = errors.New("api key has been revoked")
	ErrAPIKeyNotFound = errors.New("api key not found")

	// User errors
	ErrUserNotFound = errors.New("user not found")

	// Rate limiting
	ErrRateLimitExceeded = errors.New("daily rate limit exceeded")

	// Job queue
	ErrJobQueueFull = errors.New("job queue error")

	// Generic
	ErrInternalServer = errors.New("internal server error")
)

// ── Canonical response messages ───────────────────────────────────────────────

const (
	MsgOK           = "OK"
	MsgCreated      = "Created"
	MsgInvalidBody  = "Invalid request body"
	MsgMissingFields = "Required fields are missing"
	MsgUnauthorized = "Authentication required"
	MsgNotFound     = "Resource not found"
	MsgInternal     = "Internal server error"
)