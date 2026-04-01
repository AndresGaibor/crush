package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	appconfig "github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/fsext"
)

// LoadConfig carga la configuración de plugins desde crush.json.
// Lee la sección "plugins" del config general.
func LoadConfig(projectDir string) (*PluginConfig, error) {
	config := &PluginConfig{
		EnabledPlugins: make(map[PluginID]PluginEnable),
		PluginOptions:  make(map[PluginID]map[string]any),
	}

	// Buscar config en el proyecto y global siguiendo la misma precedencia
	// que el cargador principal de configuración.
	paths := []string{
		appconfig.GlobalConfig(),
		appconfig.GlobalConfigData(),
	}

	if found, err := fsext.Lookup(projectDir, "crush.json", ".crush/crush.json"); err == nil {
		slices.Reverse(found)
		paths = append(paths, found...)
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

		var fileConfig PluginConfig
		if err := json.Unmarshal(wrapper.Plugins, &fileConfig); err != nil {
			return nil, fmt.Errorf("parsing plugins config from %s: %w", path, err)
		}

		warnInvalidEnableRules(path, &fileConfig)
		mergePluginConfig(config, &fileConfig)
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

func mergePluginConfig(dst, src *PluginConfig) {
	if dst == nil || src == nil {
		return
	}

	if len(src.EnabledPlugins) > 0 {
		if dst.EnabledPlugins == nil {
			dst.EnabledPlugins = make(map[PluginID]PluginEnable, len(src.EnabledPlugins))
		}
		for id, enabled := range src.EnabledPlugins {
			dst.EnabledPlugins[id] = enabled
		}
	}

	if len(src.PluginOptions) > 0 {
		if dst.PluginOptions == nil {
			dst.PluginOptions = make(map[PluginID]map[string]any, len(src.PluginOptions))
		}
		for id, opts := range src.PluginOptions {
			if existing, ok := dst.PluginOptions[id]; ok {
				dst.PluginOptions[id] = mergeAnyMaps(existing, opts)
				continue
			}
			dst.PluginOptions[id] = cloneAnyMap(opts)
		}
	}

	if src.PluginsDir != "" {
		dst.PluginsDir = src.PluginsDir
	}
}

func cloneAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeAnyMaps(base, override map[string]any) map[string]any {
	if len(base) == 0 {
		return cloneAnyMap(override)
	}
	if len(override) == 0 {
		return cloneAnyMap(base)
	}

	merged := cloneAnyMap(base)
	for key, value := range override {
		if baseMap, ok := merged[key].(map[string]any); ok {
			if overrideMap, ok := value.(map[string]any); ok {
				merged[key] = mergeAnyMaps(baseMap, overrideMap)
				continue
			}
		}
		merged[key] = value
	}
	return merged
}

func warnInvalidEnableRules(path string, cfg *PluginConfig) {
	if cfg == nil || len(cfg.EnabledPlugins) == 0 {
		return
	}

	for id, raw := range cfg.EnabledPlugins {
		switch value := raw.(type) {
		case bool:
			continue
		case []string:
			for _, rule := range value {
				if err := validateVersionRequirement(rule); err != nil {
					slog.Warn("Invalid plugin version requirement",
						"path", path,
						"plugin", id,
						"requirement", rule,
						"error", err,
					)
				}
			}
		case []any:
			for _, item := range value {
				rule, ok := item.(string)
				if !ok {
					slog.Warn("Invalid plugin version requirement",
						"path", path,
						"plugin", id,
						"requirement", item,
						"error", "must be a string",
					)
					continue
				}
				if err := validateVersionRequirement(rule); err != nil {
					slog.Warn("Invalid plugin version requirement",
						"path", path,
						"plugin", id,
						"requirement", rule,
						"error", err,
					)
				}
			}
		default:
			slog.Warn("Unsupported plugin enable rule",
				"path", path,
				"plugin", id,
				"type", fmt.Sprintf("%T", raw),
			)
		}
	}
}
