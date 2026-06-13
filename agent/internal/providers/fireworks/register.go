// Package fireworks implements the Fireworks provider.
package fireworks

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("fireworks", New)
}