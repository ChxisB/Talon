// Spectre Proxy Proxy Server
//
// A local HTTP proxy that translates Anthropic Messages API requests
// to any supported provider (OpenRouter, Ollama, Gemini, etc.).
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/messaging"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
	"github.com/ChxisB/spectre-proxy/agent/internal/server"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	cfg := config.DefaultSettings()
	cfg.LoadFromEnv()

	if *configPath != "" {
		// Future: load from file
	}

	log.Printf("Spectre Proxy proxy starting (model=%s)\n", cfg.Model)

	// Start HTTP server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start messaging platforms if configured
	if cfg.DiscordBotToken != "" || cfg.TelegramBotToken != "" {
		go startMessaging(cfg, srv.Registry)
	}

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-sigCh
	log.Println("Shutting down...")
}

func startMessaging(cfg *config.Settings, registry *providers.Registry) {
	handler := messaging.NewHandler(cfg, registry)

	if cfg.DiscordBotToken != "" {
		discord := messaging.NewDiscordAdapter(cfg.DiscordBotToken, handler)
		handler.AddPlatform(discord)
	}

	if cfg.TelegramBotToken != "" {
		telegram := messaging.NewTelegramAdapter(cfg.TelegramBotToken, handler)
		handler.AddPlatform(telegram)
	}

	ctx := context.Background()
	if err := handler.Start(ctx); err != nil {
		log.Printf("Messaging error: %v", err)
	}
}
