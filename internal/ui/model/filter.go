package model

import (
	"time"

	bubble "github.com/ChxisB/talon/deps/ui/terminal/v2"
)

var lastMouseEvent time.Time

func MouseEventFilter(m bubble.Model, msg bubble.Msg) bubble.Msg {
	switch msg.(type) {
	case bubble.MouseWheelMsg, bubble.MouseMotionMsg:
		now := time.Now()
		// trackpad is sending too many requests
		if now.Sub(lastMouseEvent) < 15*time.Millisecond {
			return nil
		}
		lastMouseEvent = now
	}
	return msg
}
