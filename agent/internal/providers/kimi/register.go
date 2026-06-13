// Package kimi implements the Kimi provider.
package kimi

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("kimi", New)
}