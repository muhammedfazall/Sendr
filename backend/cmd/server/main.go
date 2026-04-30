package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/muhammedfazall/Sendr/internal/adapters/emailsender"
	"github.com/muhammedfazall/Sendr/internal/adapters/jobrepo"
	"github.com/muhammedfazall/Sendr/internal/router"
	"github.com/muhammedfazall/Sendr/internal/worker"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/muhammedfazall/Sendr/pkg/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config:", err)
	}

	pool, err := db.Connect(cfg.DBUrl)
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	defer pool.Close()
	log.Println("database connected")

	rdb, err := db.ConnectRedis(cfg.RedisUrl)
	if err != nil {
		log.Fatal("failed to connect to redis:", err)
	}
	defer rdb.Close()
	log.Println("redis connected")

	// Shared cancellable context — cancelling this stops the worker.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Worker ──────────────────────────────────────────────────────────
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	sender := emailsender.NewSendGrid(cfg.SendGridKey, cfg.FromEmail, cfg.FromName)
	jobRepo := jobrepo.New(pool)
	w := worker.New(jobRepo, sender, logger)
	go w.Run(ctx)

	// ── HTTP server ──────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router.New(cfg, pool, rdb),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("server starting on :" + cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error:", err)
		}
	}()

	// ── Graceful shutdown ────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")

	// 1. Cancel context — worker stops polling and drains in-flight jobs.
	cancel()

	// 2. Give the HTTP server 10s to finish in-flight requests.
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer httpCancel()
	if err := srv.Shutdown(httpCtx); err != nil {
		log.Println("http shutdown error:", err)
	}

	log.Println("server stopped cleanly")
}
