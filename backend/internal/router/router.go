package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/muhammedfazall/Sendr/internal/adapters/apikeyrepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/emaileventrepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/jobrepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/paymentrepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/planrepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/ratelimit"
	"github.com/muhammedfazall/Sendr/internal/adapters/templaterepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/tokenstore"
	"github.com/muhammedfazall/Sendr/internal/adapters/unsubrepo"
	"github.com/muhammedfazall/Sendr/internal/adapters/userrepo"
	"github.com/muhammedfazall/Sendr/internal/core/services"
	"github.com/muhammedfazall/Sendr/internal/handlers/apikeyhandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/authhandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/emailhandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/mehandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/paymenthandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/templatehandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/unsubhandler"
	"github.com/muhammedfazall/Sendr/internal/handlers/webhookhandler"
	"github.com/muhammedfazall/Sendr/internal/health"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/config"
)

func New(cfg *config.Config, pool *pgxpool.Pool, rdb *redis.Client, log *slog.Logger) (*chi.Mux, error) {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.StructuredLogger(log))
	r.Use(chimiddleware.Recoverer)

	// CORS — allow requests from the React dev server
	r.Use(corsMiddleware(cfg.AllowedOrigins))
	r.Use(maxBodyBytes(1 << 20))

	// Adapters
	userRepo := userrepo.New(pool)
	keyRepo := apikeyrepo.New(pool)
	jobRepo := jobrepo.New(pool)
	limiter := ratelimit.New(rdb)
	tokenStore := tokenstore.New(rdb)
	paymentRepo := paymentrepo.New(pool)
	planRepo := planrepo.New(pool)
	emailEventRepo := emaileventrepo.New(pool)
	templateRepo := templaterepo.New(pool)
	unsubRepo := unsubrepo.New(pool)

	// Core services
	authSvc, err := services.NewAuthService(userRepo, tokenStore, cfg)
	if err != nil {
		return nil, fmt.Errorf("auth service: %w", err)
	}
	apiKeySvc := services.NewAPIKeyService(keyRepo, userRepo)
	emailSvc := services.NewEmailService(apiKeySvc, jobRepo, userRepo, limiter)
	paymentSvc := services.NewPaymentService(paymentRepo, planRepo, userRepo, cfg)
	templateSvc := services.NewTemplateService(templateRepo)

	// Handlers
	authH, err := authhandler.New(authSvc, cfg)
	if err != nil {
		return nil, fmt.Errorf("auth handler: %w", err)
	}
	apikeyH := apikeyhandler.New(apiKeySvc)
	emailH := emailhandler.New(emailSvc, jobRepo, emailEventRepo, templateSvc)
	meH := mehandler.New(userRepo, limiter)
	healthH := health.NewHandler(health.NewService(pool, rdb))
	paymentH := paymenthandler.New(paymentSvc)
	webhookH := webhookhandler.New(paymentRepo, userRepo, cfg)
	sgWebhookH := webhookhandler.NewSendGridHandler(emailEventRepo, jobRepo)
	templateH := templatehandler.New(templateRepo)
	unsubH := unsubhandler.New(unsubRepo, cfg.UnsubscribeSecret)

	// Routes
	r.Get("/health", healthH.Check())
	r.Get("/plans", paymentH.ListPlans())
	r.Post("/webhooks/razorpay", webhookH.Handle())
	r.Post("/webhooks/sendgrid", sgWebhookH.Handle())

	// Auth routes — rate-limited to prevent abuse.
	r.Group(func(r chi.Router) {
		r.Use(chimiddleware.Throttle(10))
		r.Get("/auth/google", authH.Login())
		r.Get("/auth/google/callback", authH.Callback())
		r.Get("/auth/token", authH.Token())
		r.Post("/auth/refresh", authH.Refresh())
	})

	// JWT-protected routes
	jwtMW, err := middleware.JWTAuth(cfg, tokenStore)
	if err != nil {
		return nil, fmt.Errorf("JWT auth middleware: %w", err)
	}
	r.With(jwtMW).Post("/auth/logout", authH.Logout())
	r.With(jwtMW).Get("/me", meH.Get())
	r.With(jwtMW).Patch("/me", meH.Update())
	r.With(jwtMW).Post("/apikeys", apikeyH.Create())
	r.With(jwtMW).Get("/apikeys", apikeyH.List())
	r.With(jwtMW).Delete("/apikeys/{id}", apikeyH.Revoke())
	r.With(jwtMW).Get("/emails", emailH.List())
	r.With(jwtMW).Get("/emails/stats", emailH.Stats())
	r.With(jwtMW).Post("/payments/orders", paymentH.CreateOrder())
	r.With(jwtMW).Post("/payments/verify", paymentH.VerifyPayment())

	apiKey := middleware.ValidateAPIKey(pool)
	r.With(apiKey).Post("/emails/send", emailH.Send())
	r.With(apiKey).Get("/emails/{id}", emailH.GetJob())

	// Unsubscribe — no auth required; the token in the query string is the proof.
	r.Get("/unsubscribe", unsubH.ServeHTTP)
	r.Post("/unsubscribe", unsubH.ServeHTTP)

	// Template CRUD — JWT required.
	r.With(jwtMW).Post("/templates", templateH.Create())
	r.With(jwtMW).Get("/templates", templateH.List())
	r.With(jwtMW).Get("/templates/{id}", templateH.Get())
	r.With(jwtMW).Patch("/templates/{id}", templateH.Update())
	r.With(jwtMW).Delete("/templates/{id}", templateH.Delete())

	return r, nil
}

func maxBodyBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware allows the React dev server to call the API.
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))

	for _, origin := range allowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme == "" || u.Host == "" || origin == "*" {
			continue
		}
		allowed[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
