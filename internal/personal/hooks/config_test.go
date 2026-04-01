package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromFile_Valid(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hooks.json")
	content := `{
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "command": "echo 'pre bash'", "timeout": 5000}
    ],
    "PostToolUse": [
      {"matcher": "Write", "command": "echo 'post write'"}
    ],
    "SessionStart": [
      {"command": "./setup.sh"}
    ]
  }
}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	config, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Len(t, config[PreToolUse], 1)
	assert.Len(t, config[PostToolUse], 1)
	assert.Len(t, config[SessionStart], 1)
	assert.Equal(t, "Bash", config[PreToolUse][0].Matcher)
	assert.Equal(t, 5000, config[PreToolUse][0].Timeout)
}

func TestLoadFromFile_NotExists(t *testing.T) {
	t.Parallel()
	config, err := LoadFromFile("/nonexistent/hooks.json")
	require.NoError(t, err)
	assert.Nil(t, config)
}

func TestLoadFromFile_DisabledHook(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hooks.json")
	content := `{
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "command": "echo test", "enabled": false}
    ]
  }
}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	config, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Len(t, config[PreToolUse], 0) // Filtrado por enabled=false
}

func TestLoadFromFile_NoCommand(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hooks.json")
	content := `{
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash"}
    ]
  }
}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	config, err := LoadFromFile(path)
	require.NoError(t, err)
	assert.Len(t, config[PreToolUse], 0) // Filtrado por command vacío
}

func TestLoadFromDirs(t *testing.T) {
	t.Parallel()
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Dir 1: PreToolUse hook
	require.NoError(t, os.WriteFile(
		filepath.Join(dir1, "pre.json"),
		[]byte(`{"hooks": {"PreToolUse": [{"matcher": "Bash", "command": "echo pre"}]}}`),
		0o644,
	))

	// Dir 2: PostToolUse hook
	require.NoError(t, os.WriteFile(
		filepath.Join(dir2, "post.json"),
		[]byte(`{"hooks": {"PostToolUse": [{"matcher": "Write", "command": "echo post"}]}}`),
		0o644,
	))

	config, err := LoadFromDirs([]string{dir1, dir2})
	require.NoError(t, err)
	assert.Len(t, config[PreToolUse], 1)
	assert.Len(t, config[PostToolUse], 1)
}

func TestMerge(t *testing.T) {
	t.Parallel()
	base := HookConfigMap{
		PreToolUse: {{Matcher: "Bash", Command: "echo base"}},
	}
	override := HookConfigMap{
		PreToolUse:  {{Matcher: "Write", Command: "echo override"}},
		PostToolUse: {{Matcher: "Edit", Command: "echo edit"}},
	}

	merged := Merge(base, override)
	assert.Len(t, merged[PreToolUse], 2) // base + override
	assert.Len(t, merged[PostToolUse], 1)
}

func TestLoadFromBytes(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"PreToolUse": [{"matcher": "*", "command": "echo all"}],
		"Stop": [{"command": "echo done"}]
	}`)

	config, err := LoadFromBytes(data)
	require.NoError(t, err)
	assert.Len(t, config[PreToolUse], 1)
	assert.Len(t, config[Stop], 1)
}
