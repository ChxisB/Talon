// Package main is the entry point for the Talon CLI.
//
//	@title			Talon API
//	@version		1.0
//	@description	Talon is a terminal-based AI coding assistant with multi-provider support. This API is served over a Unix socket (or Windows named pipe) and provides programmatic access to workspaces, sessions, agents, LSP, MCP, and more.
//	@contact.name	Talon
//	@contact.url	https://github.com/ChxisB/talon
//	@license.name	MIT
//	@license.url	https://github.com/ChxisB/talon/blob/main/LICENSE
//	@BasePath		/v1
package main

import (
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/ChxisB/talon/internal/cmd"
	_ "github.com/ChxisB/talon/internal/dns"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	if os.Getenv("TALON_PROFILE") != "" {
		go func() {
			slog.Info("Serving pprof at localhost:6060")
			if httpErr := http.ListenAndServe("localhost:6060", nil); httpErr != nil {
				slog.Error("Failed to pprof listen", "error", httpErr)
			}
		}()
	}

	cmd.Execute()
}
