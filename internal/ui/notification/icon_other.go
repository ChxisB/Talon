//go:build !darwin

package notification

import (
	_ "embed"
)

//go:embed talon-icon-solo.png
var Icon []byte
