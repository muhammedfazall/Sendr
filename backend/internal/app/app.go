package app

import (
	"context"
	"fmt"
	"log"
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
		bus:    eventbus.New(64),
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
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
	a.worker = worker.New(jobRepo, sender, a.bus, a.logger, a.cfg.BackendURL, a.cfg.UnsubscribeSecret)
	go a.bus.Run(ctx)
	go a.worker.Run(ctx)
	log.Println("worker started")
	return nil
}

func (a *App) startServer() error {
	a.server = &http.Server{
		Addr:         ":" + a.cfg.Port,
		Handler:      router.New(a.cfg, a.pool, a.rdb),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("server starting on :" + a.cfg.Port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error:", err)
		}
	}()
	return nil
}

func (a *App) waitForShutdown() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")

	httpCtx, httpCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer httpCancel()
	if err := a.server.Shutdown(httpCtx); err != nil {
		log.Println("http shutdown error:", err)
	}

	a.pool.Close()
	a.rdb.Close()

	log.Println("server stopped cleanly")
	return nil
}