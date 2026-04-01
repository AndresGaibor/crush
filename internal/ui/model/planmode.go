package model

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/personal/planmode"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
)

func planModeState(sessionID string) (active, approved, hasPlan bool) {
	sm := planmode.PeekStateManager(sessionID)
	if sm == nil {
		return false, false, false
	}
	return sm.IsActive(), sm.IsApproved(), sm.GetPlan() != nil
}

func planModeLabel(sessionID string) string {
	active, approved, hasPlan := planModeState(sessionID)
	return planModeLabelFromState(active, approved, hasPlan)
}

func planModeLabelFromState(active, approved, hasPlan bool) string {
	switch {
	case active && approved:
		return "PLAN MODE · APPROVED"
	case active && hasPlan:
		return "PLAN MODE · REVIEW"
	case active:
		return "PLAN MODE · DRAFT"
	case hasPlan:
		return "PLAN MODE · SAVED"
	default:
		return ""
	}
}

func planModeBadge(t *styles.Styles, sessionID string) string {
	label := planModeLabel(sessionID)
	if label == "" {
		return ""
	}

	active, approved, hasPlan := planModeState(sessionID)
	sty := t.TagInfo
	switch {
	case approved:
		sty = t.TagBase.Foreground(t.FgBase).Background(t.GreenDark)
	case active:
		sty = t.TagInfo
	case hasPlan:
		sty = t.TagBase.Foreground(t.FgBase).Background(t.Yellow)
	}

	return sty.Render(label)
}

func planModeBanner(t *styles.Styles, sessionID string, width int) string {
	active, approved, hasPlan := planModeState(sessionID)
	if !active && !approved && !hasPlan {
		return ""
	}

	var parts []string
	switch {
	case active && approved:
		parts = append(parts, "Plan mode aprobado.")
		parts = append(parts, "Puedes salir cuando quieras.")
	case active && hasPlan:
		parts = append(parts, "Plan mode activo.")
		parts = append(parts, "Revisa el plan antes de continuar.")
	case active:
		parts = append(parts, "Plan mode activo.")
		parts = append(parts, "Genera un plan antes de ejecutar cambios.")
	default:
		parts = append(parts, "Plan mode guardado.")
		parts = append(parts, "Ábrelo para revisar o aprobar el plan.")
	}

	banner := strings.Join(parts, " ")
	label := planModeBadge(t, sessionID)
	if label != "" {
		banner = label + "  " + banner
	}

	if width > 0 {
		banner = ansi.Truncate(banner, width, "…")
	}
	return t.Subtle.Render(banner)
}

func planModeStatusLine(t *styles.Styles, sessionID string, modeActive bool, width int) string {
	sessionActive, approved, hasPlan := planModeState(sessionID)

	status := "plan mode off"
	if modeActive || sessionActive || approved {
		status = "plan mode on"
	} else if hasPlan {
		status = "plan mode saved"
	}

	var parts []string
	statusStyle := t.Subtle
	if modeActive || sessionActive || approved {
		statusStyle = t.Base.Foreground(t.Info).Bold(true)
	}
	switch {
	case modeActive || sessionActive || approved:
		parts = append(parts, statusStyle.Render(status))
		parts = append(parts, t.Subtle.Render("(ctrl+shift+p to toggle)"))
	case hasPlan:
		parts = append(parts, t.Base.Foreground(t.Yellow).Bold(true).Render(status))
		parts = append(parts, t.Subtle.Render("(ctrl+shift+p to toggle)"))
	default:
		parts = append(parts, statusStyle.Render(status))
		parts = append(parts, t.Subtle.Render("(ctrl+shift+p to toggle)"))
	}

	line := strings.Join(parts, " ")
	if width > 0 {
		line = ansi.Truncate(line, width, "…")
	}
	return line
}

func planModePanelView(t *styles.Styles, sessionID string, open bool, width int) string {
	if !open {
		return ""
	}

	sm := planmode.PeekStateManager(sessionID)
	title := common.Section(t, "Plan Mode", width, planModeLabel(sessionID))

	body := "Plan mode activo.\nNo hay sesión todavía.\nPresiona enter para empezar."
	if sm != nil {
		switch {
		case sm.IsActive() && sm.GetPlan() != nil:
			body = planmode.RenderPlan(sm.GetPlan())
		case sm.IsActive():
			body = "Plan mode activo.\nAún no hay un plan estructurado."
		default:
			body = "No hay un plan visible en esta sesión."
		}
	}

	panel := lipgloss.JoinVertical(lipgloss.Left, title, body)
	return t.Dialog.View.Width(width).Render(panel)
}
