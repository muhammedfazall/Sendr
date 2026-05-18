package main

import (
	"log"

	"github.com/muhammedfazall/Sendr/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatal("failed to initialize app:", err)
	}

	if err := a.Run(); err != nil {
		log.Fatal("app error:", err)
	}
}