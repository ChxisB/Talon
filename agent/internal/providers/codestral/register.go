// Package codestral implements the Codestral provider.
package codestral

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("codestral", New)
}