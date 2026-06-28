package common

import (
	bubble "github.com/ChxisB/talon/deps/ui/terminal/v2"
)

// Model represents a common interface for UI components.
type Model[T any] interface {
	Update(msg bubble.Msg) (T, bubble.Cmd)
	View() string
}
