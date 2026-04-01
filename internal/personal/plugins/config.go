package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadConfig carga la configuración de plugins desde crush.json.
// Lee la sección "plugins" del config general.
func LoadConfig(projectDir string) (*PluginConfig, error) {
	config := &PluginConfig{
		EnabledPlugins: make(map[PluginID]PluginEnable),
		PluginOptions:  make(map[PluginID]map[string]any),
	}

	// Buscar config en el proyecto y global
	paths := []string{
		filepath.Join(projectDir, ".crush", "crush.json"),
		filepath.Join(projectDir, "crush.json"),
		filepath.Join(homeDir(), ".config", "crush", "crush.json"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var wrapper struct {
			Plugins json.RawMessage `json:"plugins"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil || wrapper.Plugins == nil {
			continue
		}

		if err := json.Unmarshal(wrapper.Plugins, config); err != nil {
			return nil, fmt.Errorf("parsing plugins config from %s: %w", path, err)
		}
	}

	return config, nil
}

// LoadPluginConfig carga la config específica de un plugin (userConfig).
// Busca en .claude-plugin/config.json o similar.
func LoadPluginConfig(pluginRootDir string, pluginID PluginID) map[string]any {
	paths := []string{
		filepath.Join(pluginRootDir, "config.json"),
		filepath.Join(pluginRootDir, ".claude-plugin", "config.json"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var config map[string]any
		if err := json.Unmarshal(data, &config); err != nil {
			continue
		}
		return config
	}

	return nil
}

// SavePluginOptions guarda las opciones de un plugin en la config del proyecto.
func SavePluginOptions(projectDir string, pluginID PluginID, options map[string]any) error {
	configPath := filepath.Join(projectDir, ".crush", "crush.json")

	var existing map[string]any
	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, &existing)
	}
	if existing == nil {
		existing = make(map[string]any)
	}

	pluginsSection, _ := existing["plugins"].(map[string]any)
	if pluginsSection == nil {
		pluginsSection = make(map[string]any)
	}

	optionsSection, _ := pluginsSection["plugin_options"].(map[string]any)
	if optionsSection == nil {
		optionsSection = make(map[string]any)
	}

	optionsSection[string(pluginID)] = options
	pluginsSection["plugin_options"] = optionsSection
	existing["plugins"] = pluginsSection

	updated, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, updated, 0o644)
}
