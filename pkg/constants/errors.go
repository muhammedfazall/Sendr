package constants

import "errors"

//Sentinel errors - use errors.Is() for matching
var (
	// Auth
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("token invalid")

	// User
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")

	// API Key
	ErrAPIKeyNotFound = errors.New("api key not found")
	ErrAPIKeyRevoked  = errors.New("api key revoked")
	ErrAPIKeyInvalid  = errors.New("api key invalid")

	// Email / Send
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrInvalidPayload    = errors.New("invalid request payload")
	ErrJobQueueFull      = errors.New("job queue full")

	// Generic
	ErrNotFound       = errors.New("not found")
	ErrInternalServer = errors.New("internal server error")
)
