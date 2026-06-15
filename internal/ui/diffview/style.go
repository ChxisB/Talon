package diffview

import (
	style "github.com/ChxisB/talon/deps/style/v2"
	"github.com/ChxisB/talon/deps/util/exp/palette"
)

// LineStyle defines the styles for a given line type in the diff view.
type LineStyle struct {
	LineNumber style.Style
	Symbol     style.Style
	Code       style.Style
}

// Style defines the overall style for the diff view, including styles for
// different line types such as divider, missing, equal, insert, and delete
// lines.
type Style struct {
	DividerLine LineStyle
	MissingLine LineStyle
	EqualLine   LineStyle
	InsertLine  LineStyle
	DeleteLine  LineStyle
	Filename    LineStyle
}

// DefaultLightStyle provides a default light theme style for the diff view.
func DefaultLightStyle() Style {
	return Style{
		DividerLine: LineStyle{
			LineNumber: style.NewStyle().
				Foreground(palette.Iron).
				Background(palette.Thunder),
			Code: style.NewStyle().
				Foreground(palette.Oyster).
				Background(palette.Anchovy),
		},
		MissingLine: LineStyle{
			LineNumber: style.NewStyle().
				Background(palette.Sash),
			Code: style.NewStyle().
				Background(palette.Sash),
		},
		EqualLine: LineStyle{
			LineNumber: style.NewStyle().
				Foreground(palette.Char).
				Background(palette.Sash),
			Code: style.NewStyle().
				Foreground(palette.Pepper).
				Background(palette.Salt),
		},
		InsertLine: LineStyle{
			LineNumber: style.NewStyle().
				Foreground(palette.Turtle).
				Background(style.Color("#c8e6c9")),
			Symbol: style.NewStyle().
				Foreground(palette.Turtle).
				Background(style.Color("#e8f5e9")),
			Code: style.NewStyle().
				Foreground(palette.Pepper).
				Background(style.Color("#e8f5e9")),
		},
		DeleteLine: LineStyle{
			LineNumber: style.NewStyle().
				Foreground(palette.Cherry).
				Background(style.Color("#ffcdd2")),
			Symbol: style.NewStyle().
				Foreground(palette.Cherry).
				Background(style.Color("#ffebee")),
			Code: style.NewStyle().
				Foreground(palette.Pepper).
				Background(style.Color("#ffebee")),
		},
		Filename: LineStyle{
			LineNumber: style.NewStyle().
				Foreground(palette.Iron).
				Background(palette.Thunder),
			Code: style.NewStyle().
				Foreground(palette.Iron).
				Background(palette.Thunder),
		},
	}
}

// DefaultDarkStyle provides a default dark theme style for the diff view.
func DefaultDarkStyle() Style {
	return Style{
		DividerLine: LineStyle{
			LineNumber: style.NewStyle().
				Foreground(palette.Smoke).
				Background(palette.Sapphire),
			Code: style.NewStyle().
				Foreground(palette.Smoke).
				Background(palette.Ox),
		},
		MissingLine: LineStyle{
			LineNumber: style.NewStyle().
				Background(palette.Char),
			Code: style.NewStyle().
				Background(palette.Char),
		},
		EqualLine: LineStyle{
			LineNumber: style.NewStyle().
				Foreground(palette.Sash).
				Background(palette.Char),
			Code: style.NewStyle().
				Foreground(palette.Salt).
				Background(palette.Pepper),
		},
		InsertLine: LineStyle{
			LineNumber: style.NewStyle().
				Foreground(palette.Turtle).
				Background(style.Color("#293229")),
			Symbol: style.NewStyle().
				Foreground(palette.Turtle).
				Background(style.Color("#303a30")),
			Code: style.NewStyle().
				Foreground(palette.Salt).
				Background(style.Color("#303a30")),
		},
		DeleteLine: LineStyle{
			LineNumber: style.NewStyle().
				Foreground(palette.Cherry).
				Background(style.Color("#332929")),
			Symbol: style.NewStyle().
				Foreground(palette.Cherry).
				Background(style.Color("#3a3030")),
			Code: style.NewStyle().
				Foreground(palette.Salt).
				Background(style.Color("#3a3030")),
		},
		Filename: LineStyle{
			LineNumber: style.NewStyle().
				Foreground(palette.Smoke).
				Background(palette.Sapphire),
			Code: style.NewStyle().
				Foreground(palette.Smoke).
				Background(palette.Sapphire),
		},
	}
}
