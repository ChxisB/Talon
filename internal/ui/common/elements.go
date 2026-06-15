package common

import (
	"cmp"
	"fmt"
	"image/color"
	"strconv"
	"strings"

	style "github.com/ChxisB/talon/deps/style/v2"
	"github.com/ChxisB/talon/deps/util/ansi"
	"github.com/ChxisB/talon/internal/agent/hyper"
	"github.com/ChxisB/talon/internal/home"
	"github.com/ChxisB/talon/internal/ui/styles"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// PrettyPath formats a file path with home directory shortening and applies
// muted styling.
func PrettyPath(t *styles.Styles, path string, width int) string {
	formatted := home.Short(path)
	return t.Sidebar.WorkingDir.Width(width).Render(formatted)
}

// FormatReasoningEffort formats a reasoning effort level for display.
func FormatReasoningEffort(effort string) string {
	if effort == "xhigh" {
		return "X-High"
	}
	return cases.Title(language.English).String(effort)
}

// ModelContextInfo contains token usage and cost information for a model.
type ModelContextInfo struct {
	ContextUsed        int64
	PromptTokens       int64
	CompletionTokens   int64
	ModelContext       int64
	Cost               float64
	CostPer1MIn        float64
	CostPer1MOut       float64
	EstimatedUsage     bool
	TokenSaverEnabled  bool
	TokenSaverLevel    string  // recommended, light, moderate, aggressive
	CompressionSavings float64 // actual savings % from last output compression
	InputSavings       float64 // savings % from memory tree / input optimization
}

// ModelInfo renders model information including name, provider, reasoning
// settings, and optional context usage/cost.
func ModelInfo(t *styles.Styles, modelName, providerName, reasoningInfo string, context *ModelContextInfo, width int, hyperCredits *int) string {
	modelIcon := t.ModelInfo.Icon.Render(styles.ModelIcon)
	modelName = t.ModelInfo.Name.Render(modelName)

	// Build first line with model name and optionally provider on the same line
	var firstLine string
	if providerName != "" {
		providerInfo := t.ModelInfo.Provider.Render(fmt.Sprintf("via %s", providerName))
		modelWithProvider := fmt.Sprintf("%s %s %s", modelIcon, modelName, providerInfo)

		// Check if it fits on one line
		if style.Width(modelWithProvider) <= width {
			firstLine = modelWithProvider
		} else {
			// If it doesn't fit, put provider on next line
			firstLine = fmt.Sprintf("%s %s", modelIcon, modelName)
		}
	} else {
		firstLine = fmt.Sprintf("%s %s", modelIcon, modelName)
	}

	parts := []string{firstLine}

	// If provider didn't fit on first line, add it as second line
	if providerName != "" && !strings.Contains(firstLine, "via") {
		providerInfo := fmt.Sprintf("via %s", providerName)
		parts = append(parts, t.ModelInfo.ProviderFallback.Render(providerInfo))
	}

	if reasoningInfo != "" {
		parts = append(parts, t.ModelInfo.Reasoning.Render(reasoningInfo))
	}

	if context != nil {
		formattedInfo := formatTokensAndCost(t, context)
		parts = append(parts, style.NewStyle().PaddingLeft(2).Render(formattedInfo))
	}

	if providerName == hyper.DisplayName && hyperCredits != nil {
		hcInfo := t.ModelInfo.HypercreditIcon.Render(styles.HypercreditIcon)
		hcInfo += " "
		hcInfo += t.ModelInfo.HypercreditText.Render(fmt.Sprintf("%s Hypercredits", FormatCredits(*hyperCredits)))
		parts = append(parts, "", hcInfo)
	}

	return style.NewStyle().Width(width).Render(
		style.JoinVertical(style.Left, parts...),
	)
}

// formatTokensAndCost formats token usage and cost in a clean user-friendly layout.
// Shows token in/out, cost, and token saver status with highlighting.
func formatTokensAndCost(t *styles.Styles, ctx *ModelContextInfo) string {
	var lines []string

	// --- Context percentage line ---
	var percentage float64
	if ctx.ModelContext > 0 {
		percentage = (float64(ctx.ContextUsed) / float64(ctx.ModelContext)) * 100
	}
	percentageText := fmt.Sprintf("%d%%", int(percentage))
	if ctx.EstimatedUsage {
		percentageText = "~" + percentageText
	}
	contextPct := t.ModelInfo.TokenPercentage.Render(percentageText)
	contextTotal := t.ModelInfo.TokenCount.Render(fmt.Sprintf("(%s)", formatTokenValue(ctx.ContextUsed)))
	contextLine := fmt.Sprintf("%s %s", contextPct, contextTotal)
	if percentage > 80 {
		contextLine = fmt.Sprintf("%s %s", styles.LSPWarningIcon, contextLine)
	}
	lines = append(lines, contextLine)

	// --- Token in / Token out ---
	promptStr := t.ModelInfo.TokenCount.Render(formatTokenValue(ctx.PromptTokens))
	compStr := t.ModelInfo.TokenCount.Render(formatTokenValue(ctx.CompletionTokens))
	tokenIn := t.ModelInfo.TokenPercentage.Render("in:")
	tokenOut := t.ModelInfo.TokenPercentage.Render("out:")
	inLine := fmt.Sprintf("%s  %s", tokenIn, promptStr)
	if ctx.InputSavings > 0 {
		savingsStr := t.ModelInfo.SaverEnabled.Render(fmt.Sprintf("(saved ~%.0f%%)", ctx.InputSavings))
		inLine = fmt.Sprintf("%s %s", inLine, savingsStr)
	}
	outLine := fmt.Sprintf("%s %s", tokenOut, compStr)
	if ctx.TokenSaverEnabled && ctx.CompressionSavings > 0 {
		savingsStr := t.ModelInfo.SaverEnabled.Render(fmt.Sprintf("(saved ~%.0f%%)", ctx.CompressionSavings))
		outLine = fmt.Sprintf("%s %s", outLine, savingsStr)
	}
	lines = append(lines, inLine)
	lines = append(lines, outLine)

	// --- Cost section ---
	costStr := t.ModelInfo.Cost.Render(fmt.Sprintf("$%.4f", ctx.Cost))
	costLabel := t.ModelInfo.TokenPercentage.Render("cost:")
	lines = append(lines, fmt.Sprintf("%s %s", costLabel, costStr))

	// --- Token saver status ---
	saverLabel := t.ModelInfo.TokenPercentage.Render("token saver:")
	if ctx.TokenSaverEnabled {
		levelLabel := ctx.TokenSaverLevel
		if levelLabel == "" {
			levelLabel = "on"
		}
		saverStatus := t.ModelInfo.SaverEnabled.Render(levelLabel)
		lines = append(lines, fmt.Sprintf("%s %s", saverLabel, saverStatus))
	} else {
		saverStatus := t.ModelInfo.SaverDisabled.Render("off")
		lines = append(lines, fmt.Sprintf("%s %s", saverLabel, saverStatus))
	}

	return style.JoinVertical(style.Left, lines...)
}

// formatTokenValue formats a token count with K/M suffix.
func formatTokenValue(tokens int64) string {
	switch {
	case tokens >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	case tokens >= 1_000:
		return fmt.Sprintf("%.1fK", float64(tokens)/1_000)
	default:
		return fmt.Sprintf("%d", tokens)
	}
}

// FormatCredits formats an integer with comma separators for thousands.
func FormatCredits(n int) string {
	s := strconv.FormatInt(int64(n), 10)
	if n < 1000 {
		return s
	}
	// Calculate how many digits before the first comma.
	firstGroup := len(s) % 3
	if firstGroup == 0 {
		firstGroup = 3
	}
	var b []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && i == firstGroup {
			b = append(b, ',')
			firstGroup += 3
		}
		b = append(b, s[i])
	}
	return string(b)
}

// StatusOpts defines options for rendering a status line with icon, title,
// description, and optional extra content.
type StatusOpts struct {
	Icon             string // if empty no icon will be shown
	Title            string
	TitleColor       color.Color
	Description      string
	DescriptionColor color.Color
	ExtraContent     string // additional content to append after the description
}

// Status renders a status line with icon, title, description, and extra
// content. The description is truncated if it exceeds the available width.
func Status(t *styles.Styles, opts StatusOpts, width int) string {
	icon := opts.Icon
	title := opts.Title
	description := opts.Description

	titleColor := cmp.Or(opts.TitleColor, t.Resource.DefaultTitleFg)
	descriptionColor := cmp.Or(opts.DescriptionColor, t.Resource.DefaultDescFg)

	title = t.Resource.RowTitleBase.Foreground(titleColor).Render(title)

	if description != "" {
		extraContentWidth := style.Width(opts.ExtraContent)
		if extraContentWidth > 0 {
			extraContentWidth += 1
		}
		description = ansi.Truncate(description, width-style.Width(icon)-style.Width(title)-2-extraContentWidth, "…")
		description = t.Resource.RowDescBase.Foreground(descriptionColor).Render(description)
	}

	var content []string
	if icon != "" {
		content = append(content, icon)
	}
	content = append(content, title)
	if description != "" {
		content = append(content, description)
	}
	if opts.ExtraContent != "" {
		content = append(content, opts.ExtraContent)
	}

	return strings.Join(content, " ")
}

// Section renders a section header with a title and a horizontal line filling
// the remaining width.
func Section(t *styles.Styles, text string, width int, info ...string) string {
	char := styles.SectionSeparator
	length := style.Width(text) + 1
	remainingWidth := width - length

	var infoText string
	if len(info) > 0 {
		infoText = strings.Join(info, " ")
		if len(infoText) > 0 {
			infoText = " " + infoText
			remainingWidth -= style.Width(infoText)
		}
	}

	text = t.Section.Title.Render(text)
	if remainingWidth > 0 {
		text = text + " " + t.Section.Line.Render(strings.Repeat(char, remainingWidth)) + infoText
	}
	return text
}

// DialogTitle renders a dialog title with a decorative line filling the
// remaining width.
func DialogTitle(t *styles.Styles, title string, width int, fromColor, toColor color.Color) string {
	char := "╱"
	length := style.Width(title) + 1
	remainingWidth := width - length
	if remainingWidth > 0 {
		lines := strings.Repeat(char, remainingWidth)
		lines = styles.ApplyForegroundGrad(t.Dialog.TitleLineBase, lines, fromColor, toColor)
		title = title + " " + lines
	}
	return title
}
