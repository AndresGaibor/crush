package hooks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellExecutor_BasicCommand(t *testing.T) {
	t.Parallel()
	executor := NewShellExecutor(t.TempDir(), []string{})

	result, err := executor.Execute(context.Background(), HookConfig{
		Command: "echo 'hello from hook'",
		Timeout: 5000,
	}, HookEvent{Type: PreToolUse})

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "hello from hook")
	assert.True(t, *result.Continue)
}

func TestShellExecutor_ReadsStdin(t *testing.T) {
	t.Parallel()
	executor := NewShellExecutor(t.TempDir(), []string{})

	result, err := executor.Execute(context.Background(), HookConfig{
		Command: "cat | jq '.hook_event_name'",
		Timeout: 5000,
	}, HookEvent{Type: PreToolUse, ToolName: "Bash"})

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "PreToolUse")
}

func TestShellExecutor_BlockingExitCode(t *testing.T) {
	t.Parallel()
	executor := NewShellExecutor(t.TempDir(), []string{})

	result, err := executor.Execute(context.Background(), HookConfig{
		Command: "echo 'denied' >&2; exit 2",
		Timeout: 5000,
	}, HookEvent{Type: PreToolUse})

	require.NoError(t, err)
	assert.Equal(t, 2, result.ExitCode)
	assert.True(t, result.ShouldBlock())
}

func TestShellExecutor_JSONOutput(t *testing.T) {
	t.Parallel()
	executor := NewShellExecutor(t.TempDir(), []string{})

	jsonOutput := `{"continue": false, "reason": "blocked by policy", "decision": "deny"}`
	result, err := executor.Execute(context.Background(), HookConfig{
		Command: "echo '" + jsonOutput + "'",
		Timeout: 5000,
	}, HookEvent{Type: PreToolUse})

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.False(t, *result.Continue)
	assert.Equal(t, "deny", result.Decision)
	assert.Equal(t, "blocked by policy", result.Reason)
	assert.True(t, result.ShouldBlock())
}

func TestShellExecutor_Timeout(t *testing.T) {
	t.Parallel()
	executor := NewShellExecutor(t.TempDir(), []string{})

	start := time.Now()
	result, err := executor.Execute(context.Background(), HookConfig{
		Command: "sleep 10",
		Timeout: 500, // 500ms
	}, HookEvent{Type: PreToolUse})

	require.NoError(t, err)
	assert.NotEqual(t, 0, result.ExitCode)           // Should be non-zero (e.g., -1, 124, or similar)
	assert.Less(t, time.Since(start), 2*time.Second) // Should not last 10s
}

func TestShellExecutor_EnvironmentVariables(t *testing.T) {
	t.Parallel()
	executor := NewShellExecutor(t.TempDir(), []string{})

	result, err := executor.Execute(context.Background(), HookConfig{
		Command: "echo \"event=$CRUSH_HOOK_EVENT tool=$CRUSH_TOOL_NAME dir=$CRUSH_PROJECT_DIR\"",
		Timeout: 5000,
	}, HookEvent{Type: PreToolUse, ToolName: "Write", Cwd: "/tmp/myproject"})

	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "event=PreToolUse")
	assert.Contains(t, result.Stdout, "tool=Write")
	assert.Contains(t, result.Stdout, "dir=/tmp/myproject")
}

func TestShellExecutor_UpdatedInput(t *testing.T) {
	t.Parallel()
	executor := NewShellExecutor(t.TempDir(), []string{})

	// Hook que retorna un input modificado
	modifiedInput := map[string]string{"file_path": "/tmp/modified.txt", "content": "modified"}
	inputJSON, _ := json.Marshal(modifiedInput)
	jsonOutput := `{"continue": true, "updatedInput": ` + string(inputJSON) + `}`

	result, err := executor.Execute(context.Background(), HookConfig{
		Command: "echo '" + jsonOutput + "'",
		Timeout: 5000,
	}, HookEvent{Type: PreToolUse})

	require.NoError(t, err)
	assert.NotNil(t, result.UpdatedInput)
}

func TestShellExecutor_NonexistentCommand(t *testing.T) {
	t.Parallel()
	executor := NewShellExecutor(t.TempDir(), []string{})

	result, err := executor.Execute(context.Background(), HookConfig{
		Command: "nonexistent_command_xyz",
		Timeout: 5000,
	}, HookEvent{Type: PreToolUse})

	require.NoError(t, err) // No es error del executor, es error del comando
	assert.NotEqual(t, 0, result.ExitCode)
	assert.Contains(t, result.Stderr, "not found")
}
