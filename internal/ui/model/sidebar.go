package model

import (
	"cmp"
	"fmt"
	"image"

	style "github.com/ChxisB/talon/deps/style/v2"
	term "github.com/ChxisB/talon/deps/terminal"
	"github.com/ChxisB/talon/deps/terminal/layout"
	"github.com/ChxisB/talon/internal/tools"
	"github.com/ChxisB/talon/internal/ui/common"
	"github.com/ChxisB/talon/internal/version"
)

// modelInfo renders the current model information including reasoning
// settings and context usage/cost for the sidebar.
func (m *UI) modelInfo(width int) string {
	model := m.selectedLargeModel()
	reasoningInfo := ""
	providerName := ""

	if model != nil {
		// Get provider name first
		providerConfig, ok := m.com.Config().Providers.Get(model.ModelCfg.Provider)
		if ok {
			providerName = providerConfig.Name

			// Only check reasoning if model can reason
			if model.CatwalkCfg.CanReason {
				if len(model.CatwalkCfg.ReasoningLevels) == 0 {
					if model.ModelCfg.Think {
						reasoningInfo = "Thinking On"
					} else {
						reasoningInfo = "Thinking Off"
					}
				} else {
					reasoningEffort := cmp.Or(model.ModelCfg.ReasoningEffort, model.CatwalkCfg.DefaultReasoningEffort)
					reasoningInfo = fmt.Sprintf("Reasoning %s", common.FormatReasoningEffort(reasoningEffort))
				}
			}
		}
	}

	var modelContext *common.ModelContextInfo
	if model != nil && m.session != nil {
		// Load tools config for token saver status
		toolsCfg := tools.Load()
		tokenSaverLevel := toolsCfg.GetLevel(tools.ToolReducer)

		// Input savings from condense compression (measured, not estimated).
		// The TokenReducer reports real savings from compressing message content.
		inputSavings := m.lastInputSavings

		modelContext = &common.ModelContextInfo{
			ContextUsed:        m.session.CompletionTokens + m.session.PromptTokens,
			PromptTokens:       m.session.PromptTokens,
			CompletionTokens:   m.session.CompletionTokens,
			Cost:               m.session.Cost,
			CostPer1MIn:        model.CatwalkCfg.CostPer1MIn,
			CostPer1MOut:       model.CatwalkCfg.CostPer1MOut,
			ModelContext:       model.CatwalkCfg.ContextWindow,
			EstimatedUsage:     m.session.EstimatedUsage,
			TokenSaverEnabled:  toolsCfg.IsEnabled(tools.ToolReducer),
			TokenSaverLevel:    tokenSaverLevel,
			CompressionSavings: m.lastCompressionSavings,
			InputSavings:       inputSavings,
		}
	}
	var modelName string
	if model != nil {
		modelName = model.CatwalkCfg.Name
	}
	return common.ModelInfo(m.com.Styles, modelName, providerName, reasoningInfo, modelContext, width, m.hyperCredits)
}

// getDynamicHeightLimits will give us the num of items to show in each section based on the height
// some items are more important than others.
func getDynamicHeightLimits(availableHeight, fileCount, lspCount, mcpCount, skillCount, todoCount int) (maxFiles, maxLSPs, maxMCPs, maxSkills, maxTodos int) {
	const (
		minItemsPerSection = 2
		// Keep these high so dynamic layout uses available sidebar space
		// instead of hitting small hard limits.
		defaultMaxFilesShown    = 1000
		defaultMaxLSPsShown     = 1000
		defaultMaxMCPsShown     = 1000
		defaultMaxSkillsShown   = 1000
		defaultMaxTodosShown    = 1000
		minAvailableHeightLimit = 10
	)

	if availableHeight < minAvailableHeightLimit {
		return minItemsPerSection, minItemsPerSection, minItemsPerSection, minItemsPerSection, minItemsPerSection
	}

	maxFiles = minItemsPerSection
	maxLSPs = minItemsPerSection
	maxMCPs = minItemsPerSection
	maxSkills = minItemsPerSection
	maxTodos = minItemsPerSection

	remainingHeight := max(0, availableHeight-(minItemsPerSection*5))

	sectionValues := []*int{&maxFiles, &maxLSPs, &maxMCPs, &maxSkills, &maxTodos}
	sectionCaps := []int{defaultMaxFilesShown, defaultMaxLSPsShown, defaultMaxMCPsShown, defaultMaxSkillsShown, defaultMaxTodosShown}
	sectionNeeds := []int{max(0, fileCount-maxFiles), max(0, lspCount-maxLSPs), max(0, mcpCount-maxMCPs), max(0, skillCount-maxSkills), max(0, todoCount-maxTodos)}

	for remainingHeight > 0 {
		allocated := false
		for i, section := range sectionValues {
			if remainingHeight == 0 {
				break
			}
			if sectionNeeds[i] == 0 || *section >= sectionCaps[i] {
				continue
			}
			*section = *section + 1
			sectionNeeds[i]--
			remainingHeight--
			allocated = true
		}
		if !allocated {
			break
		}
	}

	for remainingHeight > 0 {
		allocated := false
		for i, section := range sectionValues {
			if remainingHeight == 0 {
				break
			}
			if *section >= sectionCaps[i] {
				continue
			}
			*section = *section + 1
			remainingHeight--
			allocated = true
		}
		if !allocated {
			break
		}
	}

	return maxFiles, maxLSPs, maxMCPs, maxSkills, maxTodos
}

// sidebar renders the chat sidebar containing session title, working
// directory, model info, file list, LSP status, and MCP status.
func (m *UI) drawSidebar(scr term.Screen, area term.Rectangle) {
	if m.session == nil {
		return
	}

	t := m.com.Styles
	width := area.Dx()

	title := t.Sidebar.SessionTitle.Width(width).MaxHeight(2).Render(m.session.Title)
	cwd := common.PrettyPath(t, m.com.Workspace.WorkingDir(), width)
	blocks := []string{
		title,
		"",
		cwd,
		"",
		m.modelInfo(width),
		"",
	}

	sidebarHeader := style.JoinVertical(
		style.Left,
		blocks...,
	)

	var remainingHeightArea image.Rectangle
	layout.Vertical(
		layout.Len(style.Height(sidebarHeader)),
		layout.Fill(1),
	).Split(m.layout.sidebar).Assign(new(image.Rectangle), &remainingHeightArea)
	remainingHeight := remainingHeightArea.Dy() - 6
	filesCount := 0
	for _, f := range m.sessionFiles {
		if f.Additions == 0 && f.Deletions == 0 {
			continue
		}
		filesCount++
	}

	lspsCount := len(m.lspStates)

	mcpsCount := 0
	for _, mcpCfg := range m.com.Config().MCP.Sorted() {
		if _, ok := m.mcpStates[mcpCfg.Name]; ok {
			mcpsCount++
		}
	}

	skillsCount := len(m.skillStatusItems())
	todosCount := 0
	if m.session != nil {
		todosCount = len(m.session.Todos)
	}

	maxFiles, maxLSPs, maxMCPs, maxSkills, maxTodos := getDynamicHeightLimits(remainingHeight, filesCount, lspsCount, mcpsCount, skillsCount, todosCount)

	lspSection := m.lspInfo(width, maxLSPs, true)
	mcpSection := m.mcpInfo(width, maxMCPs, true)
	skillsSection := m.skillsInfo(width, maxSkills, true)
	filesSection := m.filesInfo(m.com.Workspace.WorkingDir(), width, maxFiles, true)
	todosSection := m.todosInfo(width, maxTodos, true)

	// Collect only non-empty sections
	var sectionParts []string
	for _, s := range []string{filesSection, todosSection, lspSection, mcpSection, skillsSection} {
		if s != "" {
			sectionParts = append(sectionParts, "", s)
		}
	}

	// Build full section content
	sectionsContent := style.JoinVertical(
		style.Left,
		append([]string{sidebarHeader}, sectionParts...)...,
	)

	// Split sidebar area into sections area and version area (1 line at bottom)
	var sectionsArea, versionArea image.Rectangle
	layout.Vertical(
		layout.Fill(1),
		layout.Len(1),
	).Split(area).Assign(&sectionsArea, &versionArea)

	sectionsAreaHeight := sectionsArea.Dy()
	sectionsContentHeight := style.Height(sectionsContent)

	// Render version at absolute bottom of sidebar
	versionFooter := t.Sidebar.SessionTitle.Width(width).Render(
		fmt.Sprintf("Talon %s", version.Version),
	)
	term.NewStyledString(versionFooter).Draw(scr, versionArea)

	// Render sections with scroll support if content overflows
	if sectionsContentHeight > sectionsAreaHeight {
		// Clamp scroll offset to valid range
		maxOffset := sectionsContentHeight - sectionsAreaHeight
		m.sidebarScrollOffset = max(0, min(m.sidebarScrollOffset, maxOffset))

		// Draw content with scroll offset by shifting the draw area upward
		drawArea := image.Rect(
			sectionsArea.Min.X,
			sectionsArea.Min.Y-m.sidebarScrollOffset,
			sectionsArea.Max.X,
			sectionsArea.Max.Y,
		)
		term.NewStyledString(sectionsContent).Draw(scr, drawArea)

		// Draw scrollbar on right edge (1 column wide)
		scrollbar := common.Scrollbar(
			m.com.Styles,
			sectionsAreaHeight,
			sectionsContentHeight,
			sectionsAreaHeight,
			m.sidebarScrollOffset,
		)
		if scrollbar != "" {
			sbArea := image.Rect(
				sectionsArea.Max.X-1,
				sectionsArea.Min.Y,
				sectionsArea.Max.X,
				sectionsArea.Max.Y,
			)
			term.NewStyledString(scrollbar).Draw(scr, sbArea)
		}
	} else {
		// No scrolling needed, render normally and reset offset
		m.sidebarScrollOffset = 0
		term.NewStyledString(sectionsContent).Draw(scr, sectionsArea)
	}
}
