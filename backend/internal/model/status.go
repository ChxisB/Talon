package model

// Health represents the backend health status.
type Health struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Status represents the application readiness state.
type Status struct {
	Healthy bool   `json:"healthy"`
	Ready   bool   `json:"ready"`
	Message string `json:"message"`
}

// ErrorResponse represents a structured API error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}
