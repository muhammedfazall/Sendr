package me

import (
	"encoding/json"
	"net/http"

	"github.com/muhammedfazall/Sendr/internal/middleware"
)

func Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, _ := middleware.GetClaims(r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(claims)
	}
}
