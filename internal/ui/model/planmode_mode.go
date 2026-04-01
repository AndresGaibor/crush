package model

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/personal/planmode"
	"github.com/charmbracelet/crush/internal/ui/util"
)

type planCommand struct {
	name string
	args string
}

func parsePlanCommand(content string) (planCommand, bool) {
	trimmed := strings.TrimSpace(content)
	lower := strings.ToLower(trimmed)
	if trimmed == "" || (lower != "/plan" && !strings.HasPrefix(lower, "/plan ")) {
		return planCommand{}, false
	}

	rest := strings.TrimSpace(trimmed[len("/plan"):])
	if rest == "" {
		return planCommand{name: "enter"}, true
	}

	normalized := strings.ToLower(rest)
	switch normalized {
	case "off", "exit", "cancel", "stop", "close":
		return planCommand{name: "exit"}, true
	case "approve":
		return planCommand{name: "approve"}, true
	case "reject":
		return planCommand{name: "reject"}, true
	default:
		return planCommand{name: "enter", args: rest}, true
	}
}

func (m *UI) planModeActive() bool {
	if m.session == nil {
		return false
	}
	sm := planmode.PeekStateManager(m.session.ID)
	return sm != nil && sm.IsActive()
}

func (m *UI) ensurePlanModeSession() tea.Cmd {
	if m.hasSession() {
		return nil
	}

	newSession, err := m.com.App.Sessions.Create(context.Background(), "New Session")
	if err != nil {
		return util.ReportError(err)
	}
	if m.forceCompactMode {
		m.isCompact = true
	}
	if newSession.ID != "" {
		m.session = &newSession
		m.setState(uiChat, m.focus)
		return m.loadSession(newSession.ID)
	}
	return nil
}

func (m *UI) startPlanMode() tea.Cmd {
	var cmds []tea.Cmd

	if cmd := m.ensurePlanModeSession(); cmd != nil {
		cmds = append(cmds, cmd)
	}

	if !m.hasSession() {
		if len(cmds) == 0 {
			return nil
		}
		return tea.Batch(cmds...)
	}

	sm := planmode.GetStateManager(m.session.ID)
	if sm == nil {
		return util.ReportError(fmt.Errorf("plan mode is not initialized"))
	}
	if err := sm.Enter(); err != nil {
		if strings.Contains(err.Error(), "already active") {
			m.planPanelOpen = true
			return tea.Batch(cmds...)
		}
		return util.ReportError(err)
	}

	m.planPanelOpen = true
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *UI) exitPlanMode() tea.Cmd {
	if !m.hasSession() {
		m.planPanelOpen = false
		return nil
	}

	sm := planmode.GetStateManager(m.session.ID)
	if sm == nil || !sm.IsActive() {
		m.planPanelOpen = false
		return nil
	}

	if _, err := sm.Exit(); err != nil {
		return util.ReportError(err)
	}

	m.planPanelOpen = false
	return nil
}

func (m *UI) approvePlanMode() tea.Cmd {
	if !m.hasSession() {
		return nil
	}

	sm := planmode.GetStateManager(m.session.ID)
	if sm == nil || !sm.IsActive() {
		return nil
	}

	if err := sm.Approve(); err != nil {
		return util.ReportError(err)
	}
	if _, err := sm.Exit(); err != nil {
		return util.ReportError(err)
	}

	m.planPanelOpen = false
	return nil
}

func (m *UI) rejectPlanMode() tea.Cmd {
	if !m.hasSession() {
		return nil
	}

	sm := planmode.GetStateManager(m.session.ID)
	if sm == nil || !sm.IsActive() {
		m.planPanelOpen = false
		return nil
	}

	if err := sm.Reject("rejected from UI command"); err != nil {
		return util.ReportError(err)
	}

	m.planPanelOpen = false
	return nil
}

func (m *UI) togglePlanMode() tea.Cmd {
	if m.planModeActive() {
		return m.exitPlanMode()
	}
	return m.startPlanMode()
}
