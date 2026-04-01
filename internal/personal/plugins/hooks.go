package plugins

import (
	"encoding/json"
	"log/slog"

	personalhooks "github.com/charmbracelet/crush/internal/personal/hooks"
)

// RegisterPluginHooks registra los hooks de plugins en el sistema de hooks.
// Se llama durante la inicialización, después de cargar los plugins.
func RegisterPluginHooks(registry *Registry, hookMgr *personalhooks.Manager) {
	entries := registry.GetHooks()
	if len(entries) == 0 {
		return
	}

	// Recolectar todos los hooks de plugins
	var allPluginHooks personalhooks.HookConfigMap

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

		// Merge con la configuración existente
		allPluginHooks = personalhooks.Merge(allPluginHooks, pluginHooks)

		slog.Info("Plugin hooks registered",
			"plugin", entry.PluginID,
			"count", len(pluginHooks),
		)
	}

	// Actualizar la configuración del HookManager con los hooks de plugins
	if len(allPluginHooks) > 0 {
		hookMgr.LoadConfig(allPluginHooks)
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
