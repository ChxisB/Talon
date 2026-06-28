// Package logo renders a Talon wordmark in a stylized way.
package logo

import (
	"fmt"
	"image/color"

	style "github.com/ChxisB/talon/deps/style/v2"
	"github.com/ChxisB/talon/internal/ui/styles"
)

// letterform represents a letterform. It can be stretched horizontally by
// a given amount via the boolean argument.
type letterform func(bool) string

const diag = `╱`

// Opts are the options for rendering the talon title art.
type Opts struct {
	FieldColor   color.Color // diagonal lines
	TitleColorA  color.Color // left gradient ramp point
	TitleColorB  color.Color // right gradient ramp point
	BrandColor   color.Color // Talon text color
	VersionColor color.Color // version text color
	Width        int         // width of the rendered logo, used for truncation
	Hyper        bool        // whether it is Talon or Hypertalon

	// When true, stretch a random letterform on each render. Has no effect in
	// compact mode. Mainly for testing. In production you will want to cache
	// the stretched letterform to keep the logo from jittering on resize.
	Unstable bool
}

// Render renders the talon logo as a simple text line.
func Render(base style.Style, version string, compact bool, o Opts) string {
	return base.Render(fmt.Sprintf("talon (%s)", version))
}

// SmallRender renders a smaller version of the talon logo, suitable for
// smaller windows or sidebar usage. Note: version is not included here because
// it's shown in the header and takes up too much space in the sidebar.
func SmallRender(t *styles.Styles, width int, o Opts) string {
	return t.Logo.SmallBrand.Render("talon")
}
