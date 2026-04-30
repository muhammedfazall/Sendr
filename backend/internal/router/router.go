package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/adapters/apikeyrepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/jobrepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/ratelimit"
	"github.com/muhammedfazall/Sendr/internal/adapters/tokenstore"
	"github.com/muhammedfazall/Sendr/internal/adapters/userrepo"
	"github.com/muhammedfazall/Sendr/internal/core/services"
	"github.com/muhammedfazall/Sendr/internal/handlers/apikeyhandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/authhandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/emailhandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/mehandler"
	"github.com/muhammedfazall/Sendr/internal/health"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/redis/go-redis/v9"
)

func New(cfg *config.Config, pool *pgxpool.Pool, rdb *redis.Client) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// CORS — allow requests from the React dev server
	r.Use(corsMiddleware(cfg.FrontendURL))

	// Adapters
	userRepo := userrepo.New(pool)
	keyRepo := apikeyrepo.New(pool)
	jobRepo := jobrepo.New(pool)
	limiter := ratelimit.New(rdb)
	tokenStore := tokenstore.New(rdb)

	// Core services
	authSvc := services.NewAuthService(userRepo, tokenStore, cfg)
	apiKeySvc := services.NewApiKeyServices(keyRepo)
	emailSvc := services.NewEmailService(apiKeySvc, jobRepo, userRepo, limiter)

	// Handlers
	authH := authhandler.New(authSvc, cfg.FrontendURL)
	apikeyH := apikeyhandler.New(apiKeySvc)
	emailH := emailhandler.New(emailSvc, jobRepo)
	meH := mehandler.New(userRepo)
	healthH := health.NewHandler(health.NewService(pool, rdb))

	// Routes
	r.Get("/health", healthH.Check())

	// Auth routes — rate-limited to prevent abuse.
	r.Group(func(r chi.Router) {
		r.Use(chimiddleware.Throttle(10))
		r.Get("/auth/google", authH.Login())
		r.Get("/auth/google/callback", authH.Callback())
		r.Get("/auth/token", authH.Token())
		r.Post("/auth/refresh", authH.Refresh())
	})

	// JWT-protected routes
	jwtMW := middleware.JWTAuth(cfg.JWTPublicKeyPath)
	r.With(jwtMW).Post("/auth/logout", authH.Logout())
	r.With(jwtMW).Get("/me", meH.Get())
	r.With(jwtMW).Post("/apikeys", apikeyH.Create())
	r.With(jwtMW).Get("/apikeys", apikeyH.List())
	r.With(jwtMW).Delete("/apikeys/{id}", apikeyH.Revoke())
	r.With(jwtMW).Get("/emails", emailH.List())

	apiKey := middleware.ValidateAPIKey(pool)
	r.With(apiKey).Post("/emails/send", emailH.Send())
	r.With(apiKey).Get("/emails/{id}", emailH.GetJob())

	return r
}

// corsMiddleware allows the React dev server to call the API.
func corsMiddleware(frontendURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", frontendURL)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
