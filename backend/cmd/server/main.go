package main

import (
	"log/slog"
	"os"

	"github.com/muhammedfazall/Sendr/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		slog.Error("failed to initialize app", "err", err)
		os.Exit(1)
	}

	if err := a.Run(); err != nil {
		slog.Error("app error", "err", err)
		os.Exit(1)
	}
}