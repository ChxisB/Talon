package handler

import (
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now()

type HealthResponse struct {
	Healthy   bool   `json:"healthy"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Healthy:   true,
		Version:   "0.1.0",
		Uptime:    time.Since(startTime).Round(time.Second).String(),
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	})
}
