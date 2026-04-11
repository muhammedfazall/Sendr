package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/muhammedfazall/Sendr/internal/router"
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
	log.Println("Database connected")

	rdb, err := db.ConnectRedis(cfg.RedisUrl)
	if err != nil {
		log.Fatal("failed to connect to redis:", err)
	}
	defer rdb.Close()
	log.Println("Redis connected")

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router.New(cfg, pool, rdb),
	}

	go func() {
		log.Println("Server starting on :" + cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Server stopped cleanly")
}
