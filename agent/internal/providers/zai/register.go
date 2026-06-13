// Package zai implements the Z.ai provider.
package zai

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("zai", New)
}