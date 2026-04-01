package hooks

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// LoadFromFile carga la configuración de hooks desde un archivo JSON.
// El formato esperado es:
//
//	{
//	  "hooks": {
//	    "PreToolUse": [
//	      {"matcher": "Bash(rm *)", "command": "echo 'Warning: deleting files'", "timeout": 5000}
//	    ],
//	    "PostToolUse": [...],
//	    "SessionStart": [...],
//	    "Stop": [...]
//	  }
//	}
func LoadFromFile(path string) (HookConfigMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading hooks config: %w", err)
	}

	// Parsear el wrapper "hooks"
	var wrapper struct {
		Hooks json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parsing hooks config wrapper: %w", err)
	}
	if wrapper.Hooks == nil {
		return nil, nil
	}

	var configMap HookConfigMap
	if err := json.Unmarshal(wrapper.Hooks, &configMap); err != nil {
		return nil, fmt.Errorf("parsing hooks config map: %w", err)
	}

	// Validar y normalizar
	for event, hooks := range configMap {
		valid := false
		for _, ev := range AllHookEvents() {
			if event == ev {
				valid = true
				break
			}
		}
		if !valid {
			slog.Warn("Unknown hook event type, skipping", "event", event)
			delete(configMap, event)
			continue
		}

		// Filtrar hooks deshabilitados y validar
		var filtered []HookConfig
		for i, h := range hooks {
			if h.Enabled != nil && !*h.Enabled {
				continue
			}
			if h.Command == "" {
				slog.Warn("Hook has no command, skipping", "event", event, "index", i)
				continue
			}
			filtered = append(filtered, h)
		}
		configMap[event] = filtered
	}

	slog.Info("Hooks config loaded",
		"file", path,
		"pre_tool_use", len(configMap[PreToolUse]),
		"post_tool_use", len(configMap[PostToolUse]),
		"session_start", len(configMap[SessionStart]),
		"stop", len(configMap[Stop]),
	)

	return configMap, nil
}

// LoadFromBytes carga la configuración de hooks desde bytes JSON.
// Útil cuando los hooks vienen integrados en el Config general.
func LoadFromBytes(data []byte) (HookConfigMap, error) {
	if data == nil || len(data) == 0 {
		return nil, nil
	}

	var configMap HookConfigMap
	if err := json.Unmarshal(data, &configMap); err != nil {
		return nil, fmt.Errorf("parsing hooks from bytes: %w", err)
	}
	return configMap, nil
}

// LoadFromDirs carga hooks desde múltiples directorios de configuración.
// Los archivos se cargan en orden y se mergean (último gana).
// Directorios típicos: global (~/.config/crush/hooks/), project (.crush/hooks/)
func LoadFromDirs(dirs []string) (HookConfigMap, error) {
	merged := make(HookConfigMap)

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			slog.Warn("Failed to read hooks directory", "dir", dir, "error", err)
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !isHookFile(entry.Name()) {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			config, err := LoadFromFile(path)
			if err != nil {
				slog.Warn("Failed to load hook file", "file", path, "error", err)
				continue
			}

			// Merge: agregar hooks sin duplicar
			for event, hooks := range config {
				merged[event] = append(merged[event], hooks...)
			}
		}
	}

	return merged, nil
}

// isHookFile determina si un archivo es un archivo de configuración de hooks.
func isHookFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".json") ||
		strings.HasSuffix(lower, ".jsonc") ||
		name == "hooks.json"
}

// Merge combina dos HookConfigMap. El segundo tiene prioridad.
func Merge(base, override HookConfigMap) HookConfigMap {
	result := make(HookConfigMap)
	for event, hooks := range base {
		result[event] = append([]HookConfig{}, hooks...)
	}
	for event, hooks := range override {
		result[event] = append(result[event], hooks...)
	}
	return result
}

// ToJSON serializa la configuración de hooks a JSON.
func ToJSON(config HookConfigMap) string {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}
