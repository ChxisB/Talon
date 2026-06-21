package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/talon/backend/internal/db"
	"github.com/talon/backend/internal/handler"
)

func main() {
	// Data directory
	dataDir := os.Getenv("TALON_DATA_DIR")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".talon", "data")
	}

	// Initialize database
	if err := db.Init(dataDir); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Port
	port := os.Getenv("TALON_PORT")
	if port == "" {
		port = "8090"
	}

	mux := handler.NewRouter()
	addr := ":" + port

	log.Printf("Talon backend starting on %s", addr)
	log.Printf("Data directory: %s", dataDir)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
