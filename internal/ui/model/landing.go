package model

import (
	"image"
	"strings"

	style "github.com/ChxisB/talon/deps/style/v2"
	"github.com/ChxisB/talon/deps/terminal/layout"
	"github.com/ChxisB/talon/internal/ui/common"
	"github.com/ChxisB/talon/internal/workspace"
)

// selectedLargeModel returns the currently selected large language model from
// the agent coordinator, if one exists.
func (m *UI) selectedLargeModel() *workspace.AgentModel {
	if m.com.Workspace.AgentIsReady() {
		model := m.com.Workspace.AgentModel()
		return &model
	}
	return nil
}

// landingView renders the landing page view showing the current working
// directory, model information, and LSP/MCP status in a two-column layout.
func (m *UI) landingView() string {
	t := m.com.Styles
	width := m.layout.main.Dx()
	cwd := common.PrettyPath(t, m.com.Workspace.WorkingDir(), width)

	parts := []string{
		cwd,
	}

	parts = append(parts, "", m.modelInfo(width))
	infoSection := style.JoinVertical(style.Left, parts...)

	var remainingHeightArea image.Rectangle
	layout.Vertical(
		layout.Len(style.Height(infoSection)+1),
		layout.Fill(1),
	).Split(m.layout.main).Assign(new(image.Rectangle), &remainingHeightArea)

	mcpLspSectionWidth := min(30, (width-2)/3)

	lspSection := m.lspInfo(mcpLspSectionWidth, max(1, remainingHeightArea.Dy()), false)
	mcpSection := m.mcpInfo(mcpLspSectionWidth, max(1, remainingHeightArea.Dy()), false)
	skillsSection := m.skillsInfo(mcpLspSectionWidth, max(1, remainingHeightArea.Dy()), false)

	var sectionParts []string
	for _, s := range []string{lspSection, mcpSection, skillsSection} {
		if s != "" {
			sectionParts = append(sectionParts, s)
		}
	}
	content := strings.Join(sectionParts, "  ")
	if content == "" {
		content = " "
	}

	return style.NewStyle().
		Width(width).
		Height(m.layout.main.Dy() - 1).
		PaddingTop(1).
		Render(
			style.JoinVertical(style.Left, infoSection, "", content),
		)
}
