package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/muhammedfazall/Sendr/pkg/constants"
)

type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type okBody struct {
	Data any `json:"data"`
}

// JSON writes any value as JSON with the given status.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Success wraps data in {"data": ...} envelope.
func Success(w http.ResponseWriter, status int, data any) {
	JSON(w, status, okBody{Data: data})
}

// Error writes {"code":..., "message":...} using the explicit values.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errBody{Code: code, Message: message})
}

// FromError maps a sentinel error → HTTP status + code via constants.StatusMap.
// Falls back to 500 INTERNAL_ERROR for unknown errors.
func FromError(w http.ResponseWriter, err error) {
	for sentinel, appErr := range constants.StatusMap {
		if errors.Is(err, sentinel) {
			Error(w, appErr.HTTPStatus, appErr.Code, appErr.Message)
			return
		}
	}
	// Unmapped error — log it, return generic 500
	Error(w,
		http.StatusInternalServerError,
		"INTERNAL_ERROR",
		"an unexpected error occurred",
	)
}
