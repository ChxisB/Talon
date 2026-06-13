// Package cerebras implements the Cerebras provider.
package cerebras

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("cerebras", New)
}