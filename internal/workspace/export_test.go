package workspace

import (
	bubble "github.com/ChxisB/talon/deps/ui/terminal/v2"
)

// ConsumeEventsForTest runs the event-handling loop on the given
// channel, invoking send for translated domain messages and refreshing
// the cached workspace snapshot on ConfigChanged. Exposed for
// cross-package integration tests that cannot rely on a real
// *bubble.Program. It returns when evc is closed.
func (w *ClientWorkspace) ConsumeEventsForTest(evc <-chan any, send func(bubble.Msg)) {
	w.consumeEvents(evc, send)
}
