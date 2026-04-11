package response

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SuccessResponse struct {
	Data interface{} `json:"data"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorResponse{Code: code, Message: message})
}

func Success(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, SuccessResponse{Data: data})
}
