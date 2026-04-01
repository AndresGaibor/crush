package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Empty(t *testing.T) {
	globalConfigDir := t.TempDir()
	globalDataDir := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", globalConfigDir)
	t.Setenv("CRUSH_GLOBAL_DATA", globalDataDir)

	config, err := LoadConfig(t.TempDir())
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.True(t, config.IsEnabled("any@project")) // Default: enabled
}

func TestLoadConfig_WithPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", filepath.Join(tmpDir, "global-config"))
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "global-data"))
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

func TestLoadConfig_Precedence_ProjectOverridesGlobal(t *testing.T) {
	tmpDir := t.TempDir()
	globalConfigDir := filepath.Join(tmpDir, "global-config")
	globalDataDir := filepath.Join(tmpDir, "global-data")
	require.NoError(t, os.MkdirAll(globalConfigDir, 0o755))
	require.NoError(t, os.MkdirAll(globalDataDir, 0o755))
	t.Setenv("CRUSH_GLOBAL_CONFIG", globalConfigDir)
	t.Setenv("CRUSH_GLOBAL_DATA", globalDataDir)

	require.NoError(t, os.WriteFile(filepath.Join(globalConfigDir, "crush.json"), []byte(`{
		"plugins": {
			"enabled_plugins": {
				"shared-plugin@project": false
			}
		}
	}`), 0o644))

	projectDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "crush.json"), []byte(`{
		"plugins": {
			"enabled_plugins": {
				"shared-plugin@project": true
			}
		}
	}`), 0o644))

	config, err := LoadConfig(projectDir)
	require.NoError(t, err)
	assert.True(t, config.IsEnabled("shared-plugin@project"))
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
		{"version allowlist no match", &PluginConfig{EnabledPlugins: map[PluginID]PluginEnable{"test@project": []string{"1.2.3", "2.0.0"}}}, "test@project", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.config.IsEnabledForVersion(tt.id, "1.0.0"), tt.name)
	}
}

func TestPluginConfig_IsEnabledForVersion_Match(t *testing.T) {
	t.Parallel()

	config := &PluginConfig{
		EnabledPlugins: map[PluginID]PluginEnable{
			"test@project": []string{"^1.2.0"},
		},
	}

	assert.True(t, config.IsEnabledForVersion("test@project", "1.4.5"))
	assert.False(t, config.IsEnabledForVersion("test@project", "2.0.0"))
}

func TestPluginConfig_IsEnabledForVersion_TildeRange(t *testing.T) {
	t.Parallel()

	config := &PluginConfig{
		EnabledPlugins: map[PluginID]PluginEnable{
			"test@project": []string{"~1.2.3"},
		},
	}

	assert.True(t, config.IsEnabledForVersion("test@project", "1.2.9"))
	assert.False(t, config.IsEnabledForVersion("test@project", "1.3.0"))
}

func TestPluginConfig_IsEnabledForVersion_AndRange(t *testing.T) {
	t.Parallel()

	config := &PluginConfig{
		EnabledPlugins: map[PluginID]PluginEnable{
			"test@project": []string{">=1.2.0 <2.0.0"},
		},
	}

	assert.True(t, config.IsEnabledForVersion("test@project", "1.9.9"))
	assert.False(t, config.IsEnabledForVersion("test@project", "2.0.0"))
}

func TestPluginConfig_IsEnabledForVersion_InvalidRequirement(t *testing.T) {
	t.Parallel()

	config := &PluginConfig{
		EnabledPlugins: map[PluginID]PluginEnable{
			"test@project": []string{"not-a-version"},
		},
	}

	assert.False(t, config.IsEnabledForVersion("test@project", "1.2.3"))
}
