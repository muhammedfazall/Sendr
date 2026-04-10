package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/muhammedfazall/Sendr/internal/auth"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config:", err)
	}

	r := chi.NewRouter()

	// Global middleware — applies to every route
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Public routes — no auth needed
	r.Get("/auth/google", auth.GoogleLogin(cfg))
	r.Get("/auth/google/callback", auth.GoogleCallback(cfg))

	// Protected routes — JWT required
	r.With(middleware.JWTAuth(cfg.JWTPublicKeyPath)).Get("/me", func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value(middleware.UserClaimsKey)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(claims)
	})

	log.Println("Server starting on :8080")
	http.ListenAndServe(":"+cfg.Port, r)
}
