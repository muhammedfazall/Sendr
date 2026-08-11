package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/adapters/emailsender"
	"github.com/muhammedfazall/Sendr/internal/adapters/eventbus"
	"github.com/muhammedfazall/Sendr/internal/adapters/jobrepo"
	"github.com/muhammedfazall/Sendr/internal/router"
	"github.com/muhammedfazall/Sendr/internal/worker"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/muhammedfazall/Sendr/pkg/db"
	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg    *config.Config
	pool   *pgxpool.Pool
	rdb    *redis.Client
	server *http.Server
	bus    *eventbus.Bus
	worker *worker.Worker
	logger *slog.Logger
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config load: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevelParsed,
	}))

	pool, err := db.Connect(cfg.DBUrl)
	if err != nil {
		return nil, fmt.Errorf("db connect: %w", err)
	}

	rdb, err := db.ConnectRedis(cfg.RedisUrl)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis connect: %w", err)
	}

	return &App{
		cfg:    cfg,
		pool:   pool,
		rdb:    rdb,
		bus:    eventbus.New(64, logger.With("component", "eventbus")),
		logger: logger,
	}, nil
}

func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := a.startWorker(ctx); err != nil {
		return err
	}

	if err := a.startServer(); err != nil {
		return err
	}

	return a.waitForShutdown()
}

func (a *App) startWorker(ctx context.Context) error {
	sender, err := emailsender.NewSender(a.cfg)
	if err != nil {
		return fmt.Errorf("create email sender: %w", err)
	}
	jobRepo := jobrepo.New(a.pool)
	a.worker = worker.New(jobRepo, sender, a.bus, a.logger.With("component", "worker"), a.cfg.BackendURL, a.cfg.UnsubscribeSecret)
	go a.bus.Run(ctx)
	go a.worker.Run(ctx)
	a.logger.Info("worker started")
	return nil
}

func (a *App) startServer() error {
	mux, err := router.New(a.cfg, a.pool, a.rdb, a.logger.With("component", "http"))
	if err != nil {
		return fmt.Errorf("router setup: %w", err)
	}

	a.server = &http.Server{
		Addr:         ":" + a.cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		a.logger.Info("server starting", "port", a.cfg.Port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()
	return nil
}

func (a *App) waitForShutdown() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.logger.Info("shutting down")

	httpCtx, httpCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer httpCancel()
	if err := a.server.Shutdown(httpCtx); err != nil {
		a.logger.Error("http shutdown error", "err", err)
	}

	a.pool.Close()
	a.rdb.Close()

	a.logger.Info("server stopped cleanly")
	return nil
}