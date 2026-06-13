// Package deepseek implements the DeepSeek provider.
package deepseek

import (
	"github.com/ChxisB/spectre-proxy/agent/internal/providers"
)

func init() {
	providers.RegisterProvider("deepseek", New)
}