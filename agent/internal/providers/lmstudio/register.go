// Package lmstudio implements the LM Studio provider.
package lmstudio

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("lmstudio", New)
}