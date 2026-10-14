package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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

func New(cfg *config.Config, pool *pgxpool.Pool, rdb *redis.Client, log *slog.Logger) (*gin.Engine, error) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger(log))
	r.Use(gin.Recovery())
	r.Use(cors.New(corsConfig(cfg.AllowedOrigins)))
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
	sendgridH := webhookhandler.NewSendGridHandler(emailEventRepo, jobRepo)
	templateH := templatehandler.New(templateRepo)
	unsubH := unsubhandler.New(unsubRepo, cfg.UnsubscribeSecret)

	// Public routes
	r.GET("/health", healthH.Check)
	r.GET("/plans", paymentH.ListPlans)
	r.POST("/webhooks/razorpay", webhookH.Handle)
	r.POST("/webhooks/sendgrid", sendgridH.Handle)

	// Auth routes — throttled to blunt brute-force / login floods.
	auth := r.Group("", middleware.Throttle(10))
	auth.GET("/auth/google", authH.Login)
	auth.GET("/auth/google/callback", authH.Callback)
	auth.GET("/auth/token", authH.Token)
	auth.POST("/auth/refresh", authH.Refresh)

	// JWT-protected routes
	jwtMW, err := middleware.JWTAuth(cfg, tokenStore)
	if err != nil {
		return nil, fmt.Errorf("JWT auth middleware: %w", err)
	}
	protected := r.Group("/", jwtMW)
	protected.POST("/auth/logout", authH.Logout)
	protected.GET("/me", meH.Get)
	protected.PATCH("/me", meH.Update)
	protected.POST("/apikeys", apikeyH.Create)
	protected.GET("/apikeys", apikeyH.List)
	protected.DELETE("/apikeys/:id", apikeyH.Revoke)
	protected.GET("/emails", emailH.List)
	protected.GET("/emails/stats", emailH.Stats)
	protected.POST("/payments/orders", paymentH.CreateOrder)
	protected.POST("/payments/verify", paymentH.VerifyPayment)
	protected.POST("/templates", templateH.Create)
	protected.GET("/templates", templateH.List)
	protected.GET("/templates/:id", templateH.Get)
	protected.PATCH("/templates/:id", templateH.Update)
	protected.DELETE("/templates/:id", templateH.Delete)

	// API-key protected routes
	apiKey := middleware.ValidateAPIKey(pool)
	r.POST("/emails/send", apiKey, emailH.Send)
	r.GET("/emails/:id", apiKey, emailH.GetJob)

	// Unsubscribe — no auth required; the token in the query string is the proof.
	r.GET("/unsubscribe", unsubH.Serve)
	r.POST("/unsubscribe", unsubH.Serve)

	return r, nil
}

func maxBodyBytes(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

// corsConfig allows the configured frontend origins to call the API with credentials.
func corsConfig(allowedOrigins []string) cors.Config {
	valid := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme == "" || u.Host == "" || origin == "*" {
			continue
		}
		valid = append(valid, origin)
	}

	return cors.Config{
		AllowOrigins:     valid,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}
}
