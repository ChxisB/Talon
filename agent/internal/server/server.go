// Package server implements the HTTP server for the Spectre Proxy proxy.
// Ported from free-claude-code/api/app.py, routes.py, and services.py.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ChxisB/spectre-proxy/agent/internal/config"
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
	"github.com/ChxisB/spectre-proxy/agent/internal/router"
	"github.com/gorilla/mux"

	// Import all provider packages to trigger their init() registration
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/anthropic"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/cerebras"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/codestral"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/deepseek"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/fireworks"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/gemini"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/groq"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/kimi"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/llamacpp"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/lmstudio"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/mistral"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/nvidia_nim"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/ollama"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/opencode"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/openai"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/openrouter"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/wafer"
	_ "github.com/ChxisB/spectre-proxy/agent/internal/providers/zai"
)

// Server is the Spectre Proxy proxy HTTP server.
type Server struct {
	config      *config.Settings
	Registry    *providers.Registry // exposed for messaging system
	modelRoute  *router.ModelRouter
	router      *mux.Router
	httpServer  *http.Server
	providerCfg providers.ProviderConfig
}

// New creates a new server.
func New(cfg *config.Settings) (*Server, error) {
	s := &Server{
		config:      cfg,
		Registry:    providers.NewRegistry(),
		modelRoute:  router.NewModelRouter(cfg),
		router:      mux.NewRouter(),
		providerCfg: providers.DefaultProviderConfig(),
	}

	// Register providers
	s.registerProviders()

	// Set up routes
	s.setupRoutes()

	return s, nil
}

// registerProviders registers all available provider factories using the factory pattern.
// Provider packages self-register via init() functions.
func (s *Server) registerProviders() {
	// All providers are registered via their package init() functions.
	// The factory map is populated automatically when packages are imported.
	// We just need to ensure the registry knows about all registered factories.
	for id, factory := range providers.ProviderFactories {
		s.Registry.Register(id, factory)
	}
}

// Start begins listening and serving.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Spectre Proxy proxy starting on %s", addr)

	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}()

	return s.httpServer.ListenAndServe()
}
