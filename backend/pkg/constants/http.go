package constants

// import "net/http"

// // AppError pairs an HTTP status with a stable code string for clients.
// type AppError struct {
//   HTTPStatus int
//   Code       string
//   Message    string
// }

// // StatusMap maps sentinel errors -> HTTP response shape.
// // Use response.FromError(err) to convert - never switch on err.Error().
// var StatusMap = map[error]AppError{
//   ErrUnauthorized:      {http.StatusUnauthorized,        "UNAUTHORIZED",        "authentication required"},
//   ErrForbidden:         {http.StatusForbidden,           "FORBIDDEN",           "access denied"},
//   ErrTokenExpired:      {http.StatusUnauthorized,        "TOKEN_EXPIRED",       "token has expired"},
//   ErrTokenInvalid:      {http.StatusUnauthorized,        "TOKEN_INVALID",       "token is invalid"},
//   ErrUserNotFound:      {http.StatusNotFound,            "USER_NOT_FOUND",      "user not found"},
//   ErrUserExists:        {http.StatusConflict,            "USER_EXISTS",         "user already exists"},
//   ErrAPIKeyNotFound:    {http.StatusNotFound,            "API_KEY_NOT_FOUND",   "api key not found"},
//   ErrAPIKeyRevoked:     {http.StatusUnauthorized,        "API_KEY_REVOKED",     "api key has been revoked"},
//   ErrAPIKeyInvalid:     {http.StatusUnauthorized,        "API_KEY_INVALID",     "api key is invalid"},
//   ErrRateLimitExceeded: {http.StatusTooManyRequests,     "RATE_LIMIT_EXCEEDED", "daily send limit reached"},
//   ErrInvalidPayload:    {http.StatusBadRequest,          "INVALID_PAYLOAD",     "request body is invalid"},
//   ErrJobQueueFull:      {http.StatusServiceUnavailable,  "QUEUE_FULL",          "job queue is at capacity"},
//   ErrNotFound:          {http.StatusNotFound,            "NOT_FOUND",           "resource not found"},
//   ErrInternalServer:    {http.StatusInternalServerError, "INTERNAL_ERROR",      "an unexpected error occurred"},
// }