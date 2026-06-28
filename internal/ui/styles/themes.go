package styles

import (
	"image/color"

	"github.com/ChxisB/talon/deps/util/exp/palette"
)

// ThemeForProvider returns the Styles associated with the given provider
// ID. Unknown or empty provider IDs yield the default Default Pantera
// theme.
func ThemeForProvider(providerID string) Styles {
	switch providerID {
	case "hyper":
		return HyperTalonObsidiana()
	default:
		return DefaultPantera()
	}
}

// Default themePantera returns the Default dark theme. It matches the
// dashboard daisyUI halloween colour scheme with orange primary and
// purple secondary.
func DefaultPantera() Styles {
	return quickStyle(quickStyleOpts{
		primary:   color.RGBA{0xf9, 0x73, 0x16, 0xff}, // orange (dashboard halloween primary)
		secondary: color.RGBA{0x7c, 0x3a, 0xed, 0xff}, // purple (dashboard halloween secondary)
		accent:    palette.Bok,

		fgBase:       palette.Sash,
		fgMoreSubtle: palette.Squid,
		fgSubtle:     palette.Smoke,
		fgMostSubtle: palette.Oyster,

		onPrimary: palette.Butter,

		bgBase:         palette.Pepper,
		bgLeastVisible: palette.BBQ,
		bgLessVisible:  palette.Char,
		bgMostVisible:  palette.Iron,

		separator: palette.Char,

		destructive:       palette.Coral,
		error:             palette.Sriracha,
		warningSubtle:     palette.Zest,
		warning:           palette.Mustard,
		denied:            palette.Tang,
		busy:              palette.Citron,
		info:              palette.Malibu,
		infoMoreSubtle:    palette.Sardine,
		infoMostSubtle:    palette.Damson,
		success:           palette.Julep,
		successMoreSubtle: palette.Bok,
		successMostSubtle: palette.Guac,
	})
}

// HyperTalonObsidiana returns the HyperTalon dark theme.
func HyperTalonObsidiana() Styles {
	return quickStyle(quickStyleOpts{
		primary:   color.RGBA{0xf9, 0x73, 0x16, 0xff}, // orange (dashboard halloween primary)
		secondary: color.RGBA{0x7c, 0x3a, 0xed, 0xff}, // purple (dashboard halloween secondary)
		accent:    palette.Bok,

		fgBase:       palette.Sash,
		fgMoreSubtle: palette.Squid,
		fgSubtle:     palette.Smoke,
		fgMostSubtle: palette.Oyster,

		onPrimary: palette.Butter,

		bgBase:         palette.Pepper,
		bgLeastVisible: palette.BBQ,
		bgLessVisible:  palette.Char,
		bgMostVisible:  palette.Iron,

		separator: palette.Char,

		destructive:       palette.Coral,
		error:             palette.Sriracha,
		warningSubtle:     palette.Zest,
		warning:           palette.Mustard,
		denied:            palette.Tang,
		busy:              palette.Citron,
		info:              palette.Malibu,
		infoMoreSubtle:    palette.Sardine,
		infoMostSubtle:    palette.Damson,
		success:           palette.Julep,
		successMoreSubtle: palette.Bok,
		successMostSubtle: palette.Guac,
	})
}
