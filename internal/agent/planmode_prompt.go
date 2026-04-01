package agent

import (
	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/personal/planmode"
)

func prependPlanModeReminder(sessionID string, history []fantasy.Message) []fantasy.Message {
	reminder := planmode.SessionPrompt(sessionID)
	if reminder == "" {
		return history
	}

	return append([]fantasy.Message{fantasy.NewSystemMessage(reminder)}, history...)
}
