package main

import (
	"log"

	"github.com/muhammedfazall/Sendr/pkg/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config:", err)
	}

	_ = cfg
}