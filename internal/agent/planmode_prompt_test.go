package agent

import (
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/personal/planmode"
	"github.com/stretchr/testify/require"
)

func TestPrependPlanModeReminder(t *testing.T) {
	planmode.Reset()
	t.Cleanup(planmode.Reset)
	require.NoError(t, planmode.Init(nil))

	sm := planmode.GetStateManager("session-reminder")
	require.NoError(t, sm.Enter())
	require.NoError(t, sm.SetPlan(&planmode.Plan{
		Goal:  "Diseñar el flujo",
		Steps: []planmode.PlanStep{{Description: "Paso 1"}},
	}))

	history := []fantasy.Message{
		fantasy.NewUserMessage("Hola"),
	}

	got := prependPlanModeReminder("session-reminder", history)
	require.Len(t, got, 2)
	require.Equal(t, fantasy.MessageRoleSystem, got[0].Role)
	require.NotEmpty(t, got[0].Content)
	require.Equal(t, fantasy.MessageRoleUser, got[1].Role)
}
