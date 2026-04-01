package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseManifest_Valid(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "plugin.json")
	content := `{
		"name": "test-plugin",
		"version": "1.0.0",
		"description": "A test plugin",
		"tools": [
			{
				"name": "MyTool",
				"source": "./tools/my-tool.md"
			}
		],
		"skills": ["./skills/deploy"]
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	m, err := ParseManifest(path)
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", m.Name)
	assert.Equal(t, "1.0.0", m.Version)
	assert.Len(t, m.Tools, 1)
	assert.Equal(t, "MyTool", m.Tools[0].Name)
	assert.Len(t, m.Skills, 1)
}

func TestValidateManifest_MissingName(t *testing.T) {
	t.Parallel()
	err := ValidateManifest(&Manifest{Name: ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'name' is required")
}

func TestValidateManifest_InvalidName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		valid bool
	}{
		{"my-plugin", true},
		{"a", true},
		{"My-Plugin", false},
		{"my plugin", false},
		{"123", true},
		{strings.Repeat("a", 65), false},
	}
	for _, tt := range tests {
		err := ValidateManifest(&Manifest{Name: tt.name, Description: "test"})
		if tt.valid {
			assert.NoError(t, err, "name=%q should be valid", tt.name)
		} else {
			assert.Error(t, err, "name=%q should be invalid", tt.name)
		}
	}
}

func TestValidateManifest_InvalidVersion(t *testing.T) {
	t.Parallel()
	err := ValidateManifest(&Manifest{Name: "test", Version: "not-semver", Description: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "semver")
}

func TestValidateManifest_ToolValidation(t *testing.T) {
	t.Parallel()
	err := ValidateManifest(&Manifest{
		Name:        "test",
		Description: "test",
		Tools:       []ToolDecl{{Name: "", Source: "./t.md"}},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tools[0].name is required")

	err2 := ValidateManifest(&Manifest{
		Name:        "test",
		Description: "test",
		Tools:       []ToolDecl{{Name: "MyTool", Source: ""}},
	})
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "tools[0].source is required")
}

func TestFindManifest(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "my-plugin")
	require.NoError(t, os.MkdirAll(filepath.Join(pluginDir, ".claude-plugin"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pluginDir, ".claude-plugin", "plugin.json"),
		[]byte(`{"name": "test", "description": "test"}`),
		0o644,
	))

	path, err := FindManifest(pluginDir)
	require.NoError(t, err)
	assert.Contains(t, path, "plugin.json")
}

func TestResolvePath(t *testing.T) {
	t.Parallel()
	resolved, err := ResolvePath("/home/user/plugins/test", "./tools/my-tool.md")
	require.NoError(t, err)
	assert.Equal(t, "/home/user/plugins/test/tools/my-tool.md", resolved)

	_, err = ResolvePath("/home/user", "../etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal")
}

func TestSplitFrontmatter(t *testing.T) {
	t.Parallel()
	content := "---\nname: my-tool\ndescription: A tool\n---\n\n# Instructions\nDo stuff\n"
	fm, body := splitFrontmatter(content)
	assert.Equal(t, "my-tool", fm["name"])
	assert.Equal(t, "A tool", fm["description"])
	assert.Contains(t, body, "# Instructions")
}

func TestSplitFrontmatter_NoFrontmatter(t *testing.T) {
	t.Parallel()
	content := "# Just markdown\nNo frontmatter here"
	fm, body := splitFrontmatter(content)
	assert.Empty(t, fm)
	assert.Equal(t, content, body)
}
