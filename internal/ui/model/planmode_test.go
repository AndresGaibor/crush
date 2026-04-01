package model

import (
	"testing"

	"github.com/charmbracelet/crush/internal/personal/planmode"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestPlanModeLabelStates(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", planModeLabel(""))
	require.Equal(t, "PLAN MODE · DRAFT", planModeLabelFromState(true, false, false))
	require.Equal(t, "PLAN MODE · REVIEW", planModeLabelFromState(true, false, true))
	require.Equal(t, "PLAN MODE · APPROVED", planModeLabelFromState(true, true, true))
	require.Equal(t, "PLAN MODE · SAVED", planModeLabelFromState(false, false, true))
}

func TestPlanModeIndicatorRendersCachedState(t *testing.T) {
	planmode.Reset()
	t.Cleanup(planmode.Reset)
	require.NoError(t, planmode.Init(nil))

	sm := planmode.GetStateManager("session-1")
	require.NoError(t, sm.Enter())
	require.NoError(t, sm.SetPlan(&planmode.Plan{
		Goal: "Probar el indicador",
		Steps: []planmode.PlanStep{
			{Description: "Paso 1"},
		},
	}))

	got := planModeBadge(common.DefaultCommon(nil).Styles, "session-1")
	require.Contains(t, ansi.Strip(got), "PLAN MODE")
	require.Contains(t, ansi.Strip(got), "REVIEW")
}

func TestPlanModeStatusLineShowsOnOffState(t *testing.T) {
	styles := common.DefaultCommon(nil).Styles
	require.Contains(t, ansi.Strip(planModeStatusLine(styles, "missing", false, 80)), "plan mode off")

	planmode.Reset()
	t.Cleanup(planmode.Reset)
	require.NoError(t, planmode.Init(nil))

	sm := planmode.GetStateManager("session-2")
	require.NoError(t, sm.Enter())
	require.Contains(t, ansi.Strip(planModeStatusLine(styles, "session-2", true, 80)), "plan mode on")
}

func TestParsePlanCommand(t *testing.T) {
	t.Parallel()

	cmd, ok := parsePlanCommand("/plan")
	require.True(t, ok)
	require.Equal(t, "enter", cmd.name)

	cmd, ok = parsePlanCommand("/plan review auth flow")
	require.True(t, ok)
	require.Equal(t, "enter", cmd.name)
	require.Equal(t, "review auth flow", cmd.args)

	cmd, ok = parsePlanCommand("/plan off")
	require.True(t, ok)
	require.Equal(t, "exit", cmd.name)

	_, ok = parsePlanCommand("hello world")
	require.False(t, ok)
}
