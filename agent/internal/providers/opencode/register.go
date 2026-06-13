// Package opencode implements the OpenCode provider.
package opencode

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("opencode", NewZen)
	providers.RegisterProvider("opencode_go", NewGo)
}