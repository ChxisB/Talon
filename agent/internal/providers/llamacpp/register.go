// Package llamacpp implements the llama.cpp provider.
package llamacpp

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("llamacpp", New)
}