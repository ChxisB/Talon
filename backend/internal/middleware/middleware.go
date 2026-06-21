package middleware

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"
)

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" || isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8090")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-talon-directory, x-talon-workspace")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string) bool {
	allowed := []string{
		"http://localhost",
		"http://127.0.0.1",
		"oc://renderer",
		"tauri://localhost",
	}
	for _, a := range allowed {
		if strings.HasPrefix(origin, a) {
			return true
		}
	}
	if strings.HasSuffix(origin, ".talon.ai") {
		return true
	}
	return false
}

func WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		password := os.Getenv("TALON_SERVER_PASSWORD")
		if password == "" {
			next.ServeHTTP(w, r)
			return
		}

		username := os.Getenv("TALON_SERVER_USERNAME")
		if username == "" {
			username = "talon"
		}

		// Check token query param
		token := r.URL.Query().Get("auth_token")
		if token != "" {
			decoded, err := base64.StdEncoding.DecodeString(token)
			if err == nil {
				parts := strings.SplitN(string(decoded), ":", 2)
				if len(parts) == 2 && parts[0] == username && parts[1] == password {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Basic ") {
			decoded, err := base64.StdEncoding.DecodeString(auth[6:])
			if err == nil {
				parts := strings.SplitN(string(decoded), ":", 2)
				if len(parts) == 2 && parts[0] == username && parts[1] == password {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="Secure Area"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
