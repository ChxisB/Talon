// Package gemini implements the Google Gemini provider.
package gemini

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("gemini", New)
}