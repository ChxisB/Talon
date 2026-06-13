// Package mistral implements the Mistral provider.
package mistral

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("mistral", New)
}