package model

import (
	"cmp"
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/personal/memory"
	personalSubagents "github.com/charmbracelet/crush/internal/personal/subagents"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/logo"
	"github.com/charmbracelet/crush/internal/ui/styles"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"
	"github.com/charmbracelet/x/ansi"
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
		modelContext = &common.ModelContextInfo{
			ContextUsed:  m.session.CompletionTokens + m.session.PromptTokens,
			Cost:         m.session.Cost,
			ModelContext: model.CatwalkCfg.ContextWindow,
		}
	}
	modelName := ""
	if model != nil {
		modelName = model.CatwalkCfg.Name
	}
	return common.ModelInfo(m.com.Styles, modelName, providerName, reasoningInfo, modelContext, width)
}

// getDynamicHeightLimits returns the number of visible items for each sidebar
// section while preserving the priority order.
func getDynamicHeightLimits(availableHeight int) (maxFiles, maxMemories, maxLSPs, maxMCPs int) {
	const (
		minItemsPerSection   = 1
		defaultMaxFilesShown = 10
		defaultMaxMemories   = 4
		defaultMaxLSPsShown  = 8
		defaultMaxMCPsShown  = 8
	)

	if availableHeight <= 0 {
		return minItemsPerSection, minItemsPerSection, minItemsPerSection, minItemsPerSection
	}

	if availableHeight <= 4 {
		return minItemsPerSection, minItemsPerSection, minItemsPerSection, minItemsPerSection
	}

	maxFiles = minItemsPerSection
	maxMemories = minItemsPerSection
	maxLSPs = minItemsPerSection
	maxMCPs = minItemsPerSection

	remainingHeight := availableHeight - 4
	if remainingHeight <= 0 {
		return maxFiles, maxMemories, maxLSPs, maxMCPs
	}

	addExtra := func(current, cap int) (int, int) {
		if remainingHeight <= 0 || current >= cap {
			return current, remainingHeight
		}
		extra := min(remainingHeight, cap-current)
		current += extra
		remainingHeight -= extra
		return current, remainingHeight
	}

	maxFiles, _ = addExtra(maxFiles, defaultMaxFilesShown)
	maxMemories, _ = addExtra(maxMemories, defaultMaxMemories)
	maxLSPs, _ = addExtra(maxLSPs, defaultMaxLSPsShown)
	maxMCPs, _ = addExtra(maxMCPs, defaultMaxMCPsShown)

	return maxFiles, maxMemories, maxLSPs, maxMCPs
}

type memoryDensity int

const (
	memoryDensityCompact memoryDensity = iota
	memoryDensityMedium
	memoryDensityLarge
)

func memoryRenderDensity(width, maxItems int) memoryDensity {
	switch {
	case width < 34 || maxItems <= 1:
		return memoryDensityCompact
	case width < 54 || maxItems <= 2:
		return memoryDensityMedium
	default:
		return memoryDensityLarge
	}
}

func memorySummaryLine(stats memory.MemoryStats, width int) string {
	line := fmt.Sprintf("T:%d P:%d G:%d S:%d", stats.Total, stats.Project, stats.Global, stats.Stale)
	return ansi.Truncate(line, width, "…")
}

func memoryTagsPreview(tags []string, density memoryDensity) string {
	if len(tags) == 0 {
		return ""
	}

	limit := 0
	switch density {
	case memoryDensityMedium:
		limit = 2
	case memoryDensityLarge:
		limit = 3
	default:
		return ""
	}

	if len(tags) < limit {
		limit = len(tags)
	}

	preview := strings.Join(tags[:limit], ", ")
	if remaining := len(tags) - limit; remaining > 0 {
		preview = fmt.Sprintf("%s, +%d", preview, remaining)
	}
	return preview
}

func memoryEntryBlock(t *styles.Styles, mem memory.Memory, width int, density memoryDensity) []string {
	scope := "P"
	if mem.Scope == memory.ScopeGlobal {
		scope = "G"
	}

	icon := t.Base.Foreground(t.Primary).Render(fmt.Sprintf("[%s]", scope))
	title := ansi.Truncate(mem.ID, max(1, width-lipgloss.Width(icon)-1), "…")

	if density == memoryDensityCompact {
		return []string{common.Status(t, common.StatusOpts{
			Icon:       icon,
			Title:      title,
			TitleColor: t.Primary,
		}, width)}
	}

	tags := memoryTagsPreview(mem.Tags, density)
	lines := []string{common.Status(t, common.StatusOpts{
		Icon:       icon,
		Title:      title,
		TitleColor: t.Primary,
	}, width)}
	if tags != "" {
		indent := "    "
		tagsLine := ansi.Truncate(tags, max(1, width-lipgloss.Width(indent)), "…")
		lines = append(lines, t.Subtle.Render(indent+tagsLine))
	}
	return lines
}

func buildMemorySectionBody(t *styles.Styles, width int, maxItems int, stats memory.MemoryStats, recent []memory.Memory) []string {
	density := memoryRenderDensity(width, maxItems)
	visibleRecent := recent
	switch density {
	case memoryDensityCompact:
		if len(visibleRecent) > 1 {
			visibleRecent = visibleRecent[:1]
		}
	case memoryDensityMedium:
		if len(visibleRecent) > 2 {
			visibleRecent = visibleRecent[:2]
		}
	default:
		if len(visibleRecent) > 4 {
			visibleRecent = visibleRecent[:4]
		}
	}

	lines := make([]string, 0, maxItems+2)

	if len(visibleRecent) == 0 {
		lines = append(lines, t.Muted.Render("No memories yet"))
		return lines
	}

	for _, mem := range visibleRecent {
		lines = append(lines, memoryEntryBlock(t, mem, width, density)...)
	}

	return lines
}

func (m *UI) memoryInfo(width int, maxItems int) string {
	t := m.com.Styles
	mgr := memory.GetManager()
	title := common.Section(t, "Memory", width)
	if mgr == nil {
		return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n%s", title, t.Muted.Render("Memory not initialized")))
	}

	stats := memory.NewAger(mgr, 90*24*time.Hour, false).Stats()
	recent, err := mgr.Recent(maxItems)
	if err != nil {
		return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n%s", title, t.Muted.Render("Memory unavailable")))
	}

	body := buildMemorySectionBody(t, width, maxItems, stats, recent)
	lines := make([]string, 0, len(body)+1)
	lines = append(lines, common.Section(t, "Memory", width, t.Subtle.Render(memorySummaryLine(stats, width))))
	lines = append(lines, body...)

	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

// subagentsInfo renders the available subagents section in the sidebar.
func (m *UI) subagentsInfo(width, maxItems int, isSection bool) string {
	t := m.com.Styles
	subagents := personalSubagents.List()
	title := t.ResourceGroupTitle.Render("Subagents")
	if isSection {
		title = common.Section(t, title, width)
	}

	list := t.ResourceAdditionalText.Render("None")
	if len(subagents) > 0 {
		var rendered []string
		for _, subagent := range subagents {
			if maxItems > 0 && len(rendered) >= maxItems {
				break
			}

			icon := t.ResourceBusyIcon.String()
			if subagent.AutoDelegate {
				icon = t.ResourceOnlineIcon.String()
			}

			titleText := t.ResourceName.Render(subagent.Name)
			description := subagent.Description
			extra := ""
			if len(subagent.Tools) > 0 {
				extra = t.Subtle.Render(fmt.Sprintf("%d tools", len(subagent.Tools)))
			} else {
				extra = t.Subtle.Render("inherits tools")
			}

			rendered = append(rendered, common.Status(t, common.StatusOpts{
				Icon:         icon,
				Title:        titleText,
				Description:  description,
				ExtraContent: extra,
			}, width))
		}
		list = strings.Join(rendered, "\n")
	}

	return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n\n%s", title, list))
}

// sidebar renders the chat sidebar containing session title, working
// directory, model info, file list, LSP status, and MCP status.
func (m *UI) drawSidebar(scr uv.Screen, area uv.Rectangle) {
	if m.session == nil {
		return
	}

	const logoHeightBreakpoint = 30

	t := m.com.Styles
	width := area.Dx()
	height := area.Dy()

	title := t.Muted.Width(width).MaxHeight(2).Render(m.session.Title)
	cwd := common.PrettyPath(t, m.com.Store().WorkingDir(), width)
	sidebarLogo := m.sidebarLogo
	if height < logoHeightBreakpoint {
		sidebarLogo = logo.SmallRender(m.com.Styles, width)
	}
	blocks := []string{
		sidebarLogo,
		title,
		"",
		cwd,
		"",
		m.modelInfo(width),
		"",
	}
	if m.session != nil && !m.planPanelOpen {
		if banner := planModeBanner(m.com.Styles, m.session.ID, width); banner != "" {
			blocks = append(blocks, banner, "")
		}
	}

	sidebarHeader := lipgloss.JoinVertical(
		lipgloss.Left,
		blocks...,
	)

	_, remainingHeightArea := layout.SplitVertical(m.layout.sidebar, layout.Fixed(lipgloss.Height(sidebarHeader)))
	remainingHeight := remainingHeightArea.Dy() - 10
	if remainingHeight < 0 {
		remainingHeight = 0
	}
	maxFiles, maxMemories, maxLSPs, maxMCPs := getDynamicHeightLimits(remainingHeight)

	memorySection := m.memoryInfo(width, maxMemories)
	subagentsSection := m.subagentsInfo(width, maxMemories, true)
	lspSection := m.lspInfo(width, maxLSPs, true)
	mcpSection := m.mcpInfo(width, maxMCPs, true)
	filesSection := m.filesInfo(m.com.Store().WorkingDir(), width, maxFiles, true)

	uv.NewStyledString(
		lipgloss.NewStyle().
			MaxWidth(width).
			MaxHeight(height).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					sidebarHeader,
					memorySection,
					"",
					subagentsSection,
					"",
					filesSection,
					"",
					lspSection,
					"",
					mcpSection,
				),
			),
	).Draw(scr, area)
}
