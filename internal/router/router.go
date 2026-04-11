package router

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/apikey"
	"github.com/muhammedfazall/Sendr/internal/auth"
	"github.com/muhammedfazall/Sendr/internal/health"
	"github.com/muhammedfazall/Sendr/internal/me"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/redis/go-redis/v9"
)

func New(cfg *config.Config, pool *pgxpool.Pool, rdb *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Public
	r.Get("/health", health.Check(pool, rdb))
	r.Get("/auth/google", auth.GoogleLogin(cfg))
	r.Get("/auth/google/callback", auth.GoogleCallback(cfg, pool))

	// JWT protected
	jwt := middleware.JWTAuth(cfg.JWTPublicKeyPath)
	r.With(jwt).Get("/me", me.Get())
	r.With(jwt).Post("/apikeys", apikey.Create(pool))
	r.With(jwt).Get("/apikeys", apikey.List(pool))
	r.With(jwt).Delete("/apikeys/{id}", apikey.Revoke(pool))

	return r
}
