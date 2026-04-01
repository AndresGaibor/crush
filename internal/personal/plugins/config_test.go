package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Empty(t *testing.T) {
	t.Parallel()
	config, err := LoadConfig(t.TempDir())
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.True(t, config.IsEnabled("any@project")) // Default: enabled
}

func TestLoadConfig_WithPlugins(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".crush")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	configContent := `{
		"plugins": {
			"enabled_plugins": {
				"good-plugin@project": true,
				"bad-plugin@project": false
			},
			"plugin_options": {
				"formatter-plugin@project": {
					"formatter": "gofmt"
				}
			}
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "crush.json"), []byte(configContent), 0o644))

	config, err := LoadConfig(tmpDir)
	require.NoError(t, err)
	assert.True(t, config.IsEnabled("good-plugin@project"))
	assert.False(t, config.IsEnabled("bad-plugin@project"))
	assert.True(t, config.IsEnabled("unknown@project")) // Default

	opts := config.GetOptions("formatter-plugin@project")
	assert.NotNil(t, opts)
	assert.Equal(t, "gofmt", opts["formatter"])
}

func TestPluginConfig_IsEnabled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		config   *PluginConfig
		id       PluginID
		expected bool
	}{
		{"nil config", nil, "test@project", true},
		{"empty", &PluginConfig{}, "test@project", true},
		{"explicit true", &PluginConfig{EnabledPlugins: map[PluginID]PluginEnable{"test@project": true}}, "test@project", true},
		{"explicit false", &PluginConfig{EnabledPlugins: map[PluginID]PluginEnable{"test@project": false}}, "test@project", false},
		{"not in map", &PluginConfig{EnabledPlugins: map[PluginID]PluginEnable{"other@project": true}}, "test@project", true},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.config.IsEnabled(tt.id), tt.name)
	}
}
