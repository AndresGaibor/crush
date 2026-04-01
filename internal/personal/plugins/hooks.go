package plugins

import (
	"encoding/json"
	"log/slog"

	personalhooks "github.com/charmbracelet/crush/internal/personal/hooks"
)

// RegisterPluginHooks registra los hooks de plugins en el sistema de hooks.
// Se llama durante la inicialización, después de cargar los plugins.
func RegisterPluginHooks(registry *Registry, hookMgr *personalhooks.Manager) {
	if registry == nil || hookMgr == nil {
		return
	}

	entries := registry.GetHooks()
	if len(entries) == 0 {
		return
	}

	currentConfig := hookMgr.Config()
	mergedConfig := currentConfig

	for _, entry := range entries {
		hooksData, ok := entry.Data.(json.RawMessage)
		if !ok {
			continue
		}

		pluginHooks, err := personalhooks.LoadFromBytes(hooksData)
		if err != nil {
			slog.Warn("Failed to load plugin hooks",
				"plugin", entry.PluginID,
				"name", entry.Name,
				"error", err,
			)
			continue
		}

		// Merge conservando la configuración del usuario y sumando hooks de plugins.
		mergedConfig = personalhooks.Merge(mergedConfig, pluginHooks)

		slog.Info("Plugin hooks registered",
			"plugin", entry.PluginID,
			"count", len(pluginHooks),
		)
	}

	// Actualizar la configuración del HookManager con los hooks de plugins.
	if len(mergedConfig) > 0 {
		hookMgr.LoadConfig(mergedConfig)
	}
}

// MergePluginHooksIntoConfig combina la configuración de hooks de plugins
// con la configuración del usuario, dando prioridad a la del usuario.
func MergePluginHooksIntoConfig(
	userConfig personalhooks.HookConfigMap,
	pluginConfig personalhooks.HookConfigMap,
) personalhooks.HookConfigMap {
	// Plugins primero, luego usuario (usuario sobreescribe)
	return personalhooks.Merge(pluginConfig, userConfig)
}
