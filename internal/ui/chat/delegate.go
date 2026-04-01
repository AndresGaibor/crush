package chat

import (
	"encoding/json"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/ui/styles"
)

// NewDelegateToAgentToolMessageItem creates a new tool item for subagent delegation.
func NewDelegateToAgentToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &DelegateToolRenderContext{}, canceled)
}

type delegateToAgentInput struct {
	Agent   string `json:"agent,omitempty"`
	Task    string `json:"task,omitempty"`
	Context string `json:"context,omitempty"`
}

// DelegateToolRenderContext renders subagent delegation tool messages.
type DelegateToolRenderContext struct{}

// RenderTool implements the [ToolRenderer] interface.
func (d *DelegateToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	cappedWidth := cappedMessageWidth(width)
	if opts.IsPending() {
		return pendingTool(sty, "Subagent", opts.Anim, opts.Compact)
	}

	var params delegateToAgentInput
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "Invalid parameters"}, cappedWidth)
	}

	subagentName := strings.TrimSpace(params.Agent)
	if subagentName == "" {
		subagentName = "auto"
	}
	task := strings.TrimSpace(params.Task)
	contextText := strings.TrimSpace(params.Context)

	header := toolHeader(sty, opts.Status, "Subagent", cappedWidth, opts.Compact, "agent", subagentName)
	if opts.Compact {
		return header
	}

	var bodyParts []string
	if task != "" {
		bodyParts = append(bodyParts, lipgloss.JoinHorizontal(lipgloss.Left,
			sty.Tool.AgentTaskTag.Render("Task"),
			" ",
			sty.Tool.AgentPrompt.Width(max(0, cappedWidth-6)).Render(task),
		))
	}
	if contextText != "" {
		bodyParts = append(bodyParts, lipgloss.JoinHorizontal(lipgloss.Left,
			sty.Tool.AgentTaskTag.Render("Context"),
			" ",
			sty.Tool.AgentPrompt.Width(max(0, cappedWidth-9)).Render(contextText),
		))
	}

	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		if len(bodyParts) == 0 {
			return joinToolParts(header, earlyState)
		}
		return joinToolParts(header, lipgloss.JoinVertical(lipgloss.Left, append(bodyParts, earlyState)...))
	}

	if !opts.HasResult() || opts.Result.Content == "" {
		if len(bodyParts) == 0 {
			return header
		}
		return joinToolParts(header, lipgloss.JoinVertical(lipgloss.Left, bodyParts...))
	}

	bodyWidth := cappedWidth - toolBodyLeftPaddingTotal
	rendered := toolOutputMarkdownContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent)
	if len(bodyParts) > 0 {
		bodyParts = append(bodyParts, "")
		bodyParts = append(bodyParts, rendered)
		return joinToolParts(header, lipgloss.JoinVertical(lipgloss.Left, bodyParts...))
	}

	return joinToolParts(header, rendered)
}
