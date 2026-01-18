package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/qualys/dspm/internal/api"
	"github.com/qualys/dspm/internal/config"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	server, err := api.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	log.Printf("Starting DSPM server on %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := server.Run(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
