package list

import (
	"image"
	"strings"

	style "github.com/ChxisB/talon/deps/style/v2"
	term "github.com/ChxisB/talon/deps/terminal"
	"github.com/ChxisB/talon/internal/stringext"
)

// DefaultHighlighter is the default highlighter function that applies inverse style.
var DefaultHighlighter Highlighter = func(x, y int, c *term.Cell) *term.Cell {
	if c == nil {
		return c
	}
	c.Style.Attrs |= term.AttrReverse
	return c
}

// Highlighter represents a function that defines how to highlight text.
type Highlighter func(x, y int, c *term.Cell) *term.Cell

// HighlightContent returns the content with highlighted regions based on the specified parameters.
func HighlightContent(content string, area image.Rectangle, startLine, startCol, endLine, endCol int) string {
	var sb strings.Builder
	pos := image.Pt(-1, -1)
	HighlightBuffer(content, area, startLine, startCol, endLine, endCol, func(x, y int, c *term.Cell) *term.Cell {
		pos.X = x
		if pos.Y == -1 {
			pos.Y = y
		} else if y > pos.Y {
			sb.WriteString(strings.Repeat("\n", y-pos.Y))
			pos.Y = y
		}
		sb.WriteString(c.Content)
		return c
	})
	if sb.Len() > 0 {
		sb.WriteString("\n")
	}
	return sb.String()
}

// Highlight highlights a region of text within the given content and region.
func Highlight(content string, area image.Rectangle, startLine, startCol, endLine, endCol int, highlighter Highlighter) string {
	buf := HighlightBuffer(content, area, startLine, startCol, endLine, endCol, highlighter)
	if buf == nil {
		return content
	}
	return buf.Render()
}

// HighlightBuffer highlights a region of text within the given content and
// region, returning a [term.ScreenBuffer].
func HighlightBuffer(content string, area image.Rectangle, startLine, startCol, endLine, endCol int, highlighter Highlighter) *term.ScreenBuffer {
	content = stringext.NormalizeSpace(content)

	if startLine < 0 || startCol < 0 {
		return nil
	}

	if highlighter == nil {
		highlighter = DefaultHighlighter
	}

	width, height := area.Dx(), area.Dy()
	buf := term.NewScreenBuffer(width, height)
	styled := term.NewStyledString(content)
	styled.Draw(&buf, area)

	// Treat -1 as "end of content"
	if endLine < 0 {
		endLine = height - 1
	}
	if endCol < 0 {
		endCol = width
	}

	for y := startLine; y <= endLine && y < height; y++ {
		if y >= buf.Height() {
			break
		}

		line := buf.Line(y)

		// Determine column range for this line
		colStart := 0
		if y == startLine {
			colStart = min(startCol, len(line))
		}

		colEnd := len(line)
		if y == endLine {
			colEnd = min(endCol, len(line))
		}

		// Track last non-empty position as we go
		lastContentX := -1

		// Single pass: check content and track last non-empty position
		for x := colStart; x < colEnd; x++ {
			cell := line.At(x)
			if cell == nil {
				continue
			}

			// Update last content position if non-empty
			if cell.Content != "" && cell.Content != " " {
				lastContentX = x
			}
		}

		// Only apply highlight up to last content position
		highlightEnd := colEnd
		if lastContentX >= 0 {
			highlightEnd = lastContentX + 1
		} else if lastContentX == -1 {
			highlightEnd = colStart // No content on this line
		}

		// Apply highlight style only to cells with content
		for x := colStart; x < highlightEnd; x++ {
			if !image.Pt(x, y).In(area) {
				continue
			}
			cell := line.At(x)
			if cell != nil {
				highlighter(x, y, cell)
			}
		}
	}

	return &buf
}

// ToHighlighter converts a [style.Style] to a [Highlighter].
func ToHighlighter(lgStyle style.Style) Highlighter {
	return func(_ int, _ int, c *term.Cell) *term.Cell {
		if c != nil {
			c.Style = ToStyle(lgStyle)
		}
		return c
	}
}

// ToStyle converts an inline [style.Style] to a [term.Style].
func ToStyle(lgStyle style.Style) term.Style {
	var uvStyle term.Style

	// Colors are already color.Color
	uvStyle.Fg = lgStyle.GetForeground()
	uvStyle.Bg = lgStyle.GetBackground()

	// Build attributes using bitwise OR
	var attrs uint8

	if lgStyle.GetBold() {
		attrs |= term.AttrBold
	}

	if lgStyle.GetItalic() {
		attrs |= term.AttrItalic
	}

	if lgStyle.GetUnderline() {
		uvStyle.Underline = term.UnderlineSingle
	}

	if lgStyle.GetStrikethrough() {
		attrs |= term.AttrStrikethrough
	}

	if lgStyle.GetFaint() {
		attrs |= term.AttrFaint
	}

	if lgStyle.GetBlink() {
		attrs |= term.AttrBlink
	}

	if lgStyle.GetReverse() {
		attrs |= term.AttrReverse
	}

	uvStyle.Attrs = attrs

	return uvStyle
}

// AdjustArea adjusts the given area rectangle by subtracting margins, borders,
// and padding from the style.
func AdjustArea(area image.Rectangle, style style.Style) image.Rectangle {
	topMargin, rightMargin, bottomMargin, leftMargin := style.GetMargin()
	topBorder, rightBorder, bottomBorder, leftBorder := style.GetBorderTopSize(),
		style.GetBorderRightSize(),
		style.GetBorderBottomSize(),
		style.GetBorderLeftSize()
	topPadding, rightPadding, bottomPadding, leftPadding := style.GetPadding()

	return image.Rectangle{
		Min: image.Point{
			X: area.Min.X + leftMargin + leftBorder + leftPadding,
			Y: area.Min.Y + topMargin + topBorder + topPadding,
		},
		Max: image.Point{
			X: area.Max.X - (rightMargin + rightBorder + rightPadding),
			Y: area.Max.Y - (bottomMargin + bottomBorder + bottomPadding),
		},
	}
}
