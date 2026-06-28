package frugal

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Pressure levels
const (
	PressureNormal   = "normal"
	PressureHigh     = "high"
	PressureCritical = "critical"

	highThreshold    = 75
	criticalThreshold = 90
)

// PressureCache stores context pressure data on disk.
type PressureCache struct {
	FillPct float64 `json:"fill_pct"`
}

// GetPressureLevel returns the context pressure level based on fill percentage.
func GetPressureLevel(fillPct float64) string {
	switch {
	case fillPct >= criticalThreshold:
		return PressureCritical
	case fillPct >= highThreshold:
		return PressureHigh
	default:
		return PressureNormal
	}
}

// ShouldInject checks whether an injection should proceed given current pressure.
// priority: "essential" (always), "token-saving" (not at critical), "informational" (normal only).
func ShouldInject(fillPct float64, priority string) bool {
	switch priority {
	case "essential":
		return true
	case "token-saving":
		return GetPressureLevel(fillPct) != PressureCritical
	case "informational":
		return GetPressureLevel(fillPct) == PressureNormal
	default:
		return true
	}
}

// ReadPressureCache reads pressure data from a cache file.
func ReadPressureCache(cacheDir, sessionID string) (*PressureCache, error) {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "frugal")
	}
	name := "quality-cache.json"
	if sessionID != "" {
		name = sprintf("quality-cache-%s.json", sanitizeID(sessionID))
	}
	path := filepath.Join(cacheDir, name)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pc PressureCache
	if err := json.Unmarshal(data, &pc); err != nil {
		return nil, err
	}
	return &pc, nil
}

// WritePressureCache writes pressure data to a cache file.
func WritePressureCache(cacheDir, sessionID string, fillPct float64) error {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "frugal")
	}
	name := "quality-cache.json"
	if sessionID != "" {
		name = sprintf("quality-cache-%s.json", sanitizeID(sessionID))
	}
	path := filepath.Join(cacheDir, name)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	pc := PressureCache{FillPct: fillPct}
	data, err := json.Marshal(pc)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func sanitizeID(raw string) string {
	b := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			b = append(b, c)
		}
	}
	if len(b) == 0 {
		return "unknown"
	}
	return string(b)
}
