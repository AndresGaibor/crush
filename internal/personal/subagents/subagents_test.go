package subagents

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAndDiscover(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "code-reviewer.md")
	require.NoError(t, os.WriteFile(path, []byte(`---
name: code-reviewer
description: Reviews code changes and finds risks.
model: small
tools: [grep, view]
auto_delegate: true
visibility: private
---
Inspect the diff, point out bugs, and suggest concrete fixes.
`), 0o644))

	subagent, err := Parse(path)
	require.NoError(t, err)
	require.Equal(t, "code-reviewer", subagent.Name)
	require.Equal(t, "small", subagent.Model)
	require.Equal(t, "Inspect the diff, point out bugs, and suggest concrete fixes.", subagent.Instructions)

	discovered := Discover([]string{dir})
	require.Len(t, discovered, 1)
	require.Equal(t, "code-reviewer", discovered[0].Name)

	xml := ToPromptXML(discovered)
	require.Contains(t, xml, "<available_subagents>")
	require.Contains(t, xml, "<name>code-reviewer</name>")
	require.Contains(t, xml, "<tool>grep</tool>")
}

func TestRegistryMatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test-agent.md")
	require.NoError(t, os.WriteFile(path, []byte(`---
name: test-agent
description: Helps with testing and regression analysis.
auto_delegate: true
---
Run tests, inspect failures, and summarize the likely cause.
`), 0o644))

	reg := NewRegistry([]string{dir})
	require.NoError(t, reg.Reload(nil))

	got, ok := reg.Get("test-agent")
	require.True(t, ok)
	require.Equal(t, "test-agent", got.Name)

	match, ok := reg.Match("regression analysis")
	require.True(t, ok)
	require.Equal(t, "test-agent", match.Name)

	list := reg.All()
	require.Len(t, list, 1)
	require.Equal(t, "test-agent", list[0].Name)
}
