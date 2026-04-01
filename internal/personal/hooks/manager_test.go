package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func cleanup(t *testing.T) {
	t.Helper()
	Reset()
}

func TestManager_NewManager(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr := NewManager(t.TempDir(), HookConfigMap{
		PreToolUse: {{Command: "echo test"}},
	})
	assert.NotNil(t, mgr)
	counts := mgr.HookCount()
	assert.Equal(t, 1, counts[PreToolUse])
}

func TestManager_Fire_NoHooks(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr := NewManager(t.TempDir(), HookConfigMap{})
	result := mgr.Fire(context.Background(), HookEvent{Type: PreToolUse})
	assert.False(t, result.Fired)
	assert.Nil(t, result.Blocking)
}

func TestManager_Fire_MatchingHook(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr := NewManager(t.TempDir(), HookConfigMap{
		PreToolUse: {
			{Matcher: "Bash", Command: "echo 'hook executed'"},
		},
	})

	result := mgr.Fire(context.Background(), HookEvent{
		Type:     PreToolUse,
		ToolName: "Bash",
	})
	assert.True(t, result.Fired)
	assert.Len(t, result.Results, 1)
	assert.Nil(t, result.Blocking)
	assert.Equal(t, 0, result.Results[0].Result.ExitCode)
}

func TestManager_Fire_NoMatch(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr := NewManager(t.TempDir(), HookConfigMap{
		PreToolUse: {
			{Matcher: "Write", Command: "echo 'should not run'"},
		},
	})

	result := mgr.Fire(context.Background(), HookEvent{
		Type:     PreToolUse,
		ToolName: "Bash",
	})
	assert.False(t, result.Fired)
}

func TestManager_Fire_BlockingHook(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr := NewManager(t.TempDir(), HookConfigMap{
		PreToolUse: {
			{
				Matcher: "*",
				Command: "echo '{\"continue\": false, \"reason\": \"delete blocked\"}'",
			},
		},
	})

	result := mgr.Fire(context.Background(), HookEvent{
		Type:      PreToolUse,
		ToolName:  "Bash",
		ToolInput: "rm -rf /tmp",
	})
	assert.True(t, result.Fired)
	assert.NotNil(t, result.Blocking)
	assert.NotNil(t, result.Blocking)
	assert.Equal(t, "delete blocked", result.Blocking.Reason)
}

func TestManager_FirePreToolUse_Helper(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr := NewManager(t.TempDir(), HookConfigMap{
		PreToolUse: {
			{Matcher: "*", Command: "echo 'pre hook ran'"},
		},
	})

	result := mgr.FirePreToolUse(context.Background(), "sess1", "Write", "call-1", nil)
	// Hook executed and returned output, so we get a result with additional context
	assert.NotNil(t, result)
	assert.Contains(t, result.AdditionalContext, "pre hook ran")
}

func TestManager_FireSessionStart(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr := NewManager(t.TempDir(), HookConfigMap{
		SessionStart: {
			{Command: "echo 'welcome!'"},
		},
	})

	result := mgr.FireSessionStart(context.Background(), "sess1", "startup")
	assert.NotNil(t, result) // Tiene contexto adicional
	assert.Contains(t, result.AdditionalContext, "welcome!")
}

func TestManager_FireStop(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr := NewManager(t.TempDir(), HookConfigMap{
		Stop: {
			{Command: "echo 'done' >> /dev/null"},
		},
	})

	result := mgr.FireStop(context.Background(), "sess1", "bye")
	assert.Nil(t, result) // No blocking
}

func TestManager_OnceFlag(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	tmpFile := filepath.Join(t.TempDir(), "counter.txt")
	os.WriteFile(tmpFile, []byte("0"), 0o644)

	mgr := NewManager(t.TempDir(), HookConfigMap{
		PreToolUse: {
			{
				Matcher: "*",
				Command: "cat " + tmpFile + " | xargs -I{} sh -c 'expr {} + 1 > " + tmpFile + "'",
				Once:    true,
			},
		},
	})

	// Ejecutar 3 veces
	for i := 0; i < 3; i++ {
		mgr.Fire(context.Background(), HookEvent{Type: PreToolUse, ToolName: "Bash"})
	}

	data, _ := os.ReadFile(tmpFile)
	assert.Equal(t, "1", string(data)[:1]) // Solo se ejecutó 1 vez (check first character)
}

func TestManager_LoadConfig(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr := NewManager(t.TempDir(), HookConfigMap{})

	newConfig := HookConfigMap{
		PostToolUse: {{Matcher: "Edit", Command: "echo 'post edit'"}},
	}
	mgr.LoadConfig(newConfig)

	counts := mgr.HookCount()
	assert.Equal(t, 1, counts[PostToolUse])
	assert.Equal(t, 0, counts[PreToolUse])
}

func TestManager_MultipleHooks_FirstBlocking(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr := NewManager(t.TempDir(), HookConfigMap{
		PreToolUse: {
			{Matcher: "*", Command: "echo 'first hook'"},
			{Matcher: "*", Command: "echo '{\"continue\": false, \"reason\": \"second blocked\"}'"},
			{Matcher: "*", Command: "echo 'third hook (should not run)'"},
		},
	})

	result := mgr.Fire(context.Background(), HookEvent{
		Type:     PreToolUse,
		ToolName: "Bash",
	})
	assert.Len(t, result.Results, 2) // Solo 2: primero pasa, segundo bloquea
	assert.True(t, result.Fired)
	assert.NotNil(t, result.Blocking)
	assert.Equal(t, "second blocked", result.Blocking.Reason)
}

func TestInit_Singleton(t *testing.T) {
	t.Parallel()
	cleanup(t)
	defer cleanup(t)

	mgr1 := Init(t.TempDir(), HookConfigMap{
		PreToolUse: {{Command: "test"}},
	})
	mgr2 := GetManager()
	assert.Equal(t, mgr1, mgr2)
}
