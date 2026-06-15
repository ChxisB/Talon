package common

import (
	"slices"
	"strings"

	colorprofile "github.com/ChxisB/talon/deps/color"
	term "github.com/ChxisB/talon/deps/terminal"
	bubble "github.com/ChxisB/talon/deps/ui/terminal/v2"
	"github.com/ChxisB/talon/deps/util/ansi"
	xstr "github.com/ChxisB/talon/deps/util/exp/strings"

	"github.com/ChxisB/talon/internal/ui/notification"
)

// Capabilities define different terminal capabilities supported.
type Capabilities struct {
	// Profile is the terminal color profile used to determine how colors are
	// rendered.
	Profile colorprofile.Profile
	// Columns is the number of character columns in the terminal.
	Columns int
	// Rows is the number of character rows in the terminal.
	Rows int
	// PixelX is the width of the terminal in pixels.
	PixelX int
	// PixelY is the height of the terminal in pixels.
	PixelY int
	// KittyGraphics indicates whether the terminal supports the Kitty graphics
	// protocol.
	KittyGraphics bool
	// SixelGraphics indicates whether the terminal supports Sixel graphics.
	SixelGraphics bool
	// Env is the terminal environment variables.
	Env term.Environ
	// TerminalVersion is the terminal version string.
	TerminalVersion string
	// ReportFocusEvents indicates whether the terminal supports focus events.
	ReportFocusEvents bool
	// OSC99Notifications indicates whether the terminal supports OSC 99 notifications.
	OSC99Notifications bool
}

// Update updates the capabilities based on the given message.
func (c *Capabilities) Update(msg any) {
	switch m := msg.(type) {
	case bubble.EnvMsg:
		c.Env = term.Environ(m)
	case bubble.ColorProfileMsg:
		c.Profile = m.Profile
	case bubble.WindowSizeMsg:
		c.Columns = m.Width
		c.Rows = m.Height
	case term.PixelSizeEvent:
		c.PixelX = m.Width
		c.PixelY = m.Height
	case term.KittyGraphicsEvent:
		c.KittyGraphics = true
	case term.PrimaryDeviceAttributesEvent:
		if slices.Contains(m, 4) {
			c.SixelGraphics = true
		}
	case bubble.TerminalVersionMsg:
		c.TerminalVersion = m.Name
	case bubble.ModeReportMsg:
		switch m.Mode {
		case ansi.ModeFocusEvent:
			c.ReportFocusEvents = modeSupported(m.Value)
		}
	case term.UnknownOscEvent:
		if notification.DetectOSC99Support(string(m)) {
			c.OSC99Notifications = true
		}
	}
}

// QueryCmd returns a [bubble.Cmd] that queries the terminal for different
// capabilities.
func QueryCmd(env term.Environ) bubble.Cmd {
	var sb strings.Builder
	sb.WriteString(ansi.RequestPrimaryDeviceAttributes)
	sb.WriteString(ansi.QueryModifyOtherKeys)
	sb.WriteString(ansi.RequestModeFocusEvent)
	sb.WriteString(notification.OSC99QuerySequence())

	// Queries that should only be sent to "smart" normal terminals.
	shouldQueryFor := shouldQueryCapabilities(env)
	if shouldQueryFor {
		sb.WriteString(ansi.RequestNameVersion)
		sb.WriteString(ansi.WindowOp(14)) // Window size in pixels
		kittyReq := ansi.KittyGraphics([]byte("AAAA"), "i=31", "s=1", "v=1", "a=q", "t=d", "f=24")
		if _, isTmux := env.LookupEnv("TMUX"); isTmux {
			kittyReq = ansi.TmuxPassthrough(kittyReq)
		}
		sb.WriteString(kittyReq)
	}

	return bubble.Raw(sb.String())
}

// SupportsTrueColor returns true if the terminal supports true color.
func (c Capabilities) SupportsTrueColor() bool {
	return c.Profile == colorprofile.TrueColor
}

// SupportsKittyGraphics returns true if the terminal supports Kitty graphics.
func (c Capabilities) SupportsKittyGraphics() bool {
	return c.KittyGraphics
}

// SupportsSixelGraphics returns true if the terminal supports Sixel graphics.
func (c Capabilities) SupportsSixelGraphics() bool {
	return c.SixelGraphics
}

// CellSize returns the size of a single terminal cell in pixels.
func (c Capabilities) CellSize() (width, height int) {
	if c.Columns == 0 || c.Rows == 0 {
		return 0, 0
	}
	return c.PixelX / c.Columns, c.PixelY / c.Rows
}

func modeSupported(v ansi.ModeSetting) bool {
	return v.IsSet() || v.IsReset()
}

// kittyTerminals defines terminals supporting querying capabilities.
var kittyTerminals = []string{"alacritty", "ghostty", "kitty", "rio", "wezterm"}

func shouldQueryCapabilities(env term.Environ) bool {
	const osVendorTypeApple = "Apple"
	termType := env.Getenv("TERM")
	termProg, okTermProg := env.LookupEnv("TERM_PROGRAM")
	_, okSSHTTY := env.LookupEnv("SSH_TTY")
	if okTermProg && strings.Contains(termProg, osVendorTypeApple) {
		return false
	}
	return (!okTermProg && !okSSHTTY) ||
		(!strings.Contains(termProg, osVendorTypeApple) && !okSSHTTY) ||
		// Terminals that do support XTVERSION.
		xstr.ContainsAnyOf(termType, kittyTerminals...)
}
