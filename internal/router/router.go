package router

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/health"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/internal/module/apikey"
	"github.com/muhammedfazall/Sendr/internal/module/auth"
	"github.com/muhammedfazall/Sendr/internal/user"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/redis/go-redis/v9"
)

func New(cfg *config.Config, pool *pgxpool.Pool, rdb *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// — Build dependency chain: repository → service → handler —

	// Auth
	authRepo := auth.NewRepository(pool)
	authService := auth.NewService(authRepo, cfg)
	authHandler := auth.NewHandler(authService)

	// API Keys
	apikeyRepo := apikey.NewRepository(pool)
	apikeyService := apikey.NewService(apikeyRepo)
	apikeyHandler := apikey.NewHandler(apikeyService)

	// Health
	healthService := health.NewService(pool, rdb)
	healthHandler := health.NewHandler(healthService)

	// Me
	userService := user.NewService()
	userHandler := user.NewHandler(userService)

	// — Routes —

	// Public
	r.Get("/health", healthHandler.Check())
	r.Get("/auth/google", authHandler.GoogleLogin())
	r.Get("/auth/google/callback", authHandler.GoogleCallback())

	// JWT protected
	jwt := middleware.JWTAuth(cfg.JWTPublicKeyPath)
	r.With(jwt).Get("/me", userHandler.Get())
	r.With(jwt).Post("/apikeys", apikeyHandler.Create())
	r.With(jwt).Get("/apikeys", apikeyHandler.List())
	r.With(jwt).Delete("/apikeys/{id}", apikeyHandler.Revoke())

	return r
}
