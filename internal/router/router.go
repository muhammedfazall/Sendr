package router

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/muhammedfazall/Sendr/internal/adapters/apikeyrepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/jobrepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/ratelimit"
	"github.com/muhammedfazall/Sendr/internal/adapters/userrepo"
	"github.com/muhammedfazall/Sendr/internal/core/services"
	"github.com/muhammedfazall/Sendr/internal/handlers/apikeyhandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/authhandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/emailhandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/mehandler"
	"github.com/muhammedfazall/Sendr/internal/health"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/config"
)

func New(cfg *config.Config, pool *pgxpool.Pool, rdb *redis.Client) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// ── Adapters ─────────────────────────────────────────────────────────
	userRepo := userrepo.New(pool)
	keyRepo  := apikeyrepo.New(pool)
	jobRepo  := jobrepo.New(pool)
	limiter  := ratelimit.New(rdb)

	// ── Core services ─────────────────────────────────────────────────────
	authSvc   := services.NewAuthService(userRepo, cfg)
	apiKeySvc := services.NewApiKeyServices(keyRepo)
	emailSvc  := services.NewEmailService(apiKeySvc, jobRepo, userRepo, limiter)

	// ── Handlers ──────────────────────────────────────────────────────────
	authH   := authhandler.New(authSvc)
	apikeyH := apikeyhandler.New(apiKeySvc)
	emailH  := emailhandler.New(emailSvc, jobRepo)
	meH     := mehandler.New(userRepo)
	healthH := health.NewHandler(health.NewService(pool, rdb))

	// ── Routes ────────────────────────────────────────────────────────────

	// Public
	r.Get("/health", healthH.Check())
	r.Get("/auth/google", authH.Login())
	r.Get("/auth/google/callback", authH.Callback())

	// JWT-protected: user dashboard
	jwt := middleware.JWTAuth(cfg.JWTPublicKeyPath)
	r.With(jwt).Get("/me", meH.Get())
	r.With(jwt).Post("/apikeys", apikeyH.Create())
	r.With(jwt).Get("/apikeys", apikeyH.List())
	r.With(jwt).Delete("/apikeys/{id}", apikeyH.Revoke())

	// API-key-protected: developer send endpoint
	apiKey := middleware.ValidateAPIKey(pool)
	r.With(apiKey).Post("/emails/send", emailH.Send())
	r.With(apiKey).Get("/emails/{id}", emailH.GetJob())

	return r
}