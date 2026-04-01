package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_DiscoverPlugins(t *testing.T) {
	t.Parallel()

	// Crear estructura de plugins falsa
	tmpDir := t.TempDir()

	// Plugin 1
	p1Dir := filepath.Join(tmpDir, "plugin-one")
	require.NoError(t, os.MkdirAll(filepath.Join(p1Dir, ".claude-plugin"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(p1Dir, ".claude-plugin", "plugin.json"),
		[]byte(`{"name": "plugin-one", "description": "First plugin"}`),
		0o644,
	))

	// Plugin 2
	p2Dir := filepath.Join(tmpDir, "plugin-two")
	require.NoError(t, os.MkdirAll(p2Dir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(p2Dir, "plugin.json"),
		[]byte(`{"name": "plugin-two", "description": "Second plugin", "version": "2.0.0"}`),
		0o644,
	))

	// Directorio vacío (no es plugin)
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "not-a-plugin"), 0o755))

	loader := NewLoader(tmpDir, &PluginConfig{})
	plugins, err := loader.discoverPlugins(tmpDir, ScopeProject)
	require.NoError(t, err)
	assert.Len(t, plugins, 2)

	names := []string{plugins[0].Name, plugins[1].Name}
	assert.Contains(t, names, "plugin-one")
	assert.Contains(t, names, "plugin-two")
}

func TestLoader_LoadAll(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Crear plugin válido
	pDir := filepath.Join(tmpDir, "my-plugin")
	require.NoError(t, os.MkdirAll(filepath.Join(pDir, ".claude-plugin"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pDir, ".claude-plugin", "plugin.json"),
		[]byte(`{
			"name": "my-plugin",
			"description": "Test plugin",
			"version": "1.0.0",
			"tools": [{"name": "MyTool", "source": "./tools/my-tool.md"}]
		}`),
		0o644,
	))

	loader := NewLoader(tmpDir, &PluginConfig{})
	plugins, err := loader.LoadAll()
	require.NoError(t, err)
	require.Len(t, plugins, 1)

	p := plugins[0]
	assert.Equal(t, "my-plugin", p.Name)
	assert.Equal(t, "1.0.0", p.Version)
	assert.Equal(t, StatusEnabled, p.Status)
	assert.NotNil(t, p.Manifest)
	assert.Len(t, p.Manifest.Tools, 1)
}

func TestLoader_DisabledPlugin(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	pDir := filepath.Join(tmpDir, "disabled-plugin")
	require.NoError(t, os.MkdirAll(pDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pDir, "plugin.json"),
		[]byte(`{"name": "disabled-plugin", "description": "Test"}`),
		0o644,
	))

	config := &PluginConfig{
		EnabledPlugins: map[PluginID]PluginEnable{
			"disabled-plugin@project": false,
		},
	}

	loader := NewLoader(tmpDir, config)
	plugins, err := loader.LoadAll()
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	assert.Equal(t, StatusDisabled, plugins[0].Status)
}

func TestLoader_InvalidManifest(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	pDir := filepath.Join(tmpDir, "bad-plugin")
	require.NoError(t, os.MkdirAll(pDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(pDir, "plugin.json"),
		[]byte(`{"name": "bad-plugin", "description": "Bad plugin", "tools": [{"name": "invalid-tool-name", "source": "./t.md"}]}`),
		0o644,
	))

	loader := NewLoader(tmpDir, &PluginConfig{})
	plugins, err := loader.LoadAll()
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	assert.Equal(t, StatusError, plugins[0].Status)
	assert.Contains(t, plugins[0].Error, "PascalCase")
}
