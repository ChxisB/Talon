package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

type cacheEntry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

func main() {
	dsn := os.Getenv("CACHE_DSN")
	if dsn == "" {
		dsn = "/data/cache.db"
	}

	var err error
	db, err = sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS cache (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			expires_at TEXT NOT NULL
		)
	`); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /get/{key}", handleGet)
	mux.HandleFunc("PUT /set/{key}", handleSet)
	mux.HandleFunc("DELETE /del/{key}", handleDel)
	mux.HandleFunc("POST /purge", handlePurge)

	port := os.Getenv("CACHE_PORT")
	if port == "" {
		port = "8083"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Cache server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	var entry cacheEntry
	err := db.QueryRowContext(r.Context(),
		"SELECT key, value, created_at, expires_at FROM cache WHERE key = ? AND expires_at > datetime('now')",
		key,
	).Scan(&entry.Key, &entry.Value, &entry.CreatedAt, &entry.ExpiresAt)

	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func handleSet(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	var body struct {
		Value string `json:"value"`
		TTL   string `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	ttl := 24 * time.Hour
	if body.TTL != "" {
		var err error
		ttl, err = time.ParseDuration(body.TTL)
		if err != nil {
			http.Error(w, "invalid ttl", http.StatusBadRequest)
			return
		}
	}

	if body.Value == "" {
		http.Error(w, "missing value", http.StatusBadRequest)
		return
	}

	ttlSec := fmt.Sprintf("+%d seconds", int(ttl.Seconds()))
	_, err := db.ExecContext(r.Context(),
		`INSERT INTO cache (key, value, created_at, expires_at)
		 VALUES (?, ?, datetime('now'), datetime('now', ?))
		 ON CONFLICT(key) DO UPDATE SET value = ?, expires_at = datetime('now', ?)`,
		key, body.Value, ttlSec, body.Value, ttlSec,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "set", "key": key, "ttl": ttl.String()})
}

func handleDel(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	_, err := db.ExecContext(r.Context(), "DELETE FROM cache WHERE key = ?", key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "key": key})
}

func handlePurge(w http.ResponseWriter, r *http.Request) {
	res, err := db.ExecContext(r.Context(), "DELETE FROM cache WHERE expires_at <= datetime('now')")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()

	db.ExecContext(r.Context(), "PRAGMA optimize")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "purged", "removed": n})
}
