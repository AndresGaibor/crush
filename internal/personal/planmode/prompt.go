package planmode

import (
	"strings"
)

// SessionPrompt devuelve una instrucción de sistema para el modo plan.
func SessionPrompt(sessionID string) string {
	sm := PeekStateManager(sessionID)
	if sm == nil || !sm.IsActive() {
		return ""
	}

	var lines []string
	lines = append(lines, "<system_reminder>Plan mode is active.")
	lines = append(lines, "Only refine, review, or restate the plan.")
	lines = append(lines, "Do not execute file changes, shell commands, or other state-changing actions while plan mode is active.")

	if plan := sm.GetPlan(); plan != nil {
		lines = append(lines, "")
		lines = append(lines, "Current plan:")
		lines = append(lines, RenderPlan(plan))
	} else {
		lines = append(lines, "")
		lines = append(lines, "Create a structured plan before execution.")
	}

	lines = append(lines, "</system_reminder>")
	return strings.Join(lines, "\n")
}
