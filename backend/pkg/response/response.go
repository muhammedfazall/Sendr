package response

import (
	"encoding/json"
	"net/http"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type successBody struct {
	Data any `json:"data"`
}

// JSON writes status + any value as JSON. Used for ad-hoc shapes.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Error writes a structured {code, message} error response.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorBody{Code: code, Message: message})
}

// Success wraps data in {data: ...} and writes 200.
func Success(w http.ResponseWriter, status int, data any) {
	JSON(w, status, successBody{Data: data})
}