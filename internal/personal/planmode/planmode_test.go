package planmode

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildToolsNames(t *testing.T) {
	t.Parallel()

	tools := BuildTools()
	require.Len(t, tools, 2)

	assert.Equal(t, enterPlanModeToolName, tools[0].Info().Name)
	assert.Equal(t, exitPlanModeToolName, tools[1].Info().Name)
}

func TestStateManagerLifecycle(t *testing.T) {
	sm := NewStateManager()

	require.False(t, sm.IsActive())
	require.NoError(t, sm.Enter())
	require.True(t, sm.IsActive())

	plan := &Plan{
		Goal: "Implementar plan mode",
		Steps: []PlanStep{
			{Description: "Diseñar el flujo"},
			{Description: "Implementar el cambio", Dependencies: []int{1}},
		},
		Considerations: []string{"Mantener el estado por sesión."},
		AffectedFiles:  []string{"internal/personal/planmode/tools.go"},
	}

	require.NoError(t, sm.SetPlan(plan))

	got := sm.GetPlan()
	require.NotNil(t, got)
	require.Len(t, got.Steps, 2)
	assert.Equal(t, 1, got.Steps[0].Number)
	assert.Equal(t, StepPending, got.Steps[0].Status)
	assert.Equal(t, 2, got.Steps[1].Number)
	assert.Equal(t, StepPending, got.Steps[1].Status)

	next := got.NextPendingStep()
	require.NotNil(t, next)
	assert.Equal(t, 1, next.Number)

	require.NoError(t, sm.Approve())

	approved, err := sm.Exit()
	require.NoError(t, err)
	require.NotNil(t, approved)
	assert.False(t, sm.IsActive())
	assert.Equal(t, StepInProgress, approved.Steps[0].Status)

	require.NoError(t, sm.Enter())
	require.True(t, sm.IsActive())
	require.NotNil(t, sm.GetPlan())
	assert.Equal(t, "Implementar plan mode", sm.GetPlan().Goal)
}

func TestStateManagerRejectClearsState(t *testing.T) {
	sm := NewStateManager()

	require.NoError(t, sm.Enter())
	require.NoError(t, sm.SetPlan(&Plan{
		Goal:  "Revisar rechazo",
		Steps: []PlanStep{{Description: "Paso 1"}},
	}))

	require.NoError(t, sm.Reject("Cambios solicitados por el usuario"))

	assert.False(t, sm.IsActive())
	assert.False(t, sm.IsApproved())
	assert.Nil(t, sm.GetPlan())
}

func TestRegistryReturnsSameManagerPerSession(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	require.NoError(t, Init(nil))

	first := GetStateManager("session-a")
	require.NotNil(t, first)
	second := GetStateManager("session-a")
	require.Same(t, first, second)

	other := GetStateManager("session-b")
	require.NotNil(t, other)
	require.NotSame(t, first, other)
}

func TestPersistentStateIsLoadedAfterReset(t *testing.T) {
	db := openTestDB(t)
	Reset()
	require.NoError(t, Init(db))

	sm := GetStateManager("session-persist")
	require.NoError(t, sm.Enter())
	require.NoError(t, sm.SetPlan(&Plan{
		ID:   "plan-1",
		Goal: "Persistir el plan",
		Steps: []PlanStep{
			{Description: "Paso 1"},
			{Description: "Paso 2", Dependencies: []int{1}},
		},
	}))
	require.NoError(t, sm.Approve())
	approved, err := sm.Exit()
	require.NoError(t, err)
	require.NotNil(t, approved)
	require.Equal(t, StepInProgress, approved.Steps[0].Status)

	Reset()
	require.NoError(t, Init(db))

	loaded := GetStateManager("session-persist")
	require.NotNil(t, loaded)
	assert.False(t, loaded.IsActive())
	assert.False(t, loaded.IsApproved())
	require.NotNil(t, loaded.GetPlan())
	assert.Equal(t, "Persistir el plan", loaded.GetPlan().Goal)
	assert.Equal(t, StepInProgress, loaded.GetPlan().Steps[0].Status)
}

func TestSessionPromptReflectsActivePlanMode(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	require.NoError(t, Init(nil))

	sm := GetStateManager("session-reminder")
	require.NoError(t, sm.Enter())
	require.NoError(t, sm.SetPlan(&Plan{
		Goal:  "Diseñar el flujo",
		Steps: []PlanStep{{Description: "Paso 1"}},
	}))

	prompt := SessionPrompt("session-reminder")
	require.NotEmpty(t, prompt)
	require.Contains(t, prompt, "Plan mode is active.")
	require.Contains(t, prompt, "Current plan:")
	require.Contains(t, prompt, "Diseñar el flujo")
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), fmt.Sprintf("%s.db", t.Name()))
	db, err := sql.Open("sqlite", "file:"+dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
