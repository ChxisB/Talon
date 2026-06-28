package format

import (
	"context"
	"errors"
	"fmt"
	"os"

	bubble "github.com/ChxisB/talon/deps/ui/terminal/v2"
	"github.com/ChxisB/talon/deps/util/ansi"
	"github.com/ChxisB/talon/internal/ui/anim"
)

// Spinner wraps the bubbles spinner for non-interactive mode
type Spinner struct {
	done chan struct{}
	prog *bubble.Program
}

type model struct {
	cancel context.CancelFunc
	anim   *anim.Anim
}

func (m model) Init() bubble.Cmd  { return m.anim.Start() }
func (m model) View() bubble.View { return bubble.NewView(m.anim.Render()) }

// Update implements bubble.Model.
func (m model) Update(msg bubble.Msg) (bubble.Model, bubble.Cmd) {
	switch msg := msg.(type) {
	case bubble.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancel()
			return m, bubble.Quit
		}
	case anim.StepMsg:
		cmd := m.anim.Animate(msg)
		return m, cmd
	}
	return m, nil
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(ctx context.Context, cancel context.CancelFunc, animSettings anim.Settings) *Spinner {
	m := model{
		anim:   anim.New(animSettings),
		cancel: cancel,
	}

	p := bubble.NewProgram(m, bubble.WithOutput(os.Stderr), bubble.WithContext(ctx))

	return &Spinner{
		prog: p,
		done: make(chan struct{}, 1),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		_, err := s.prog.Run()
		// ensures line is cleared
		fmt.Fprint(os.Stderr, ansi.EraseEntireLine)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, bubble.ErrInterrupted) {
			fmt.Fprintf(os.Stderr, "Error running spinner: %v\n", err)
		}
	}()
}

// Stop ends the spinner animation
func (s *Spinner) Stop() {
	s.prog.Quit()
	<-s.done
}
