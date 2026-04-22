package main

import (
	"log"
	"net/http"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/api"
	"github.com/snowfallx-bot/SnowPanel/backend/internal/config"
)

func main() {
	cfg := config.Load()
	server := &http.Server{
		Addr:         cfg.Address(),
		Handler:      api.NewRouter(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	log.Printf("backend listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("backend stopped unexpectedly: %v", err)
	}
}
