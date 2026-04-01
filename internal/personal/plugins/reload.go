package plugins

import (
	"fmt"
	"log/slog"
)

// Reload vuelve a descubrir y cargar los plugins desde disco.
// El reemplazo del estado interno es atómico.
func (m *Manager) Reload() error {
	if err := m.ensureProjectDir(); err != nil {
		return err
	}

	_, _, _, projectDir := m.snapshot()

	config, err := LoadConfig(projectDir)
	if err != nil {
		slog.Warn("Failed to reload plugin config, using defaults", "error", err)
		config = &PluginConfig{}
	}

	loader := NewLoader(projectDir, config)
	plugins, err := loader.LoadAll()
	if err != nil {
		return fmt.Errorf("reloading plugins: %w", err)
	}

	registry := NewRegistry()
	registry.PopulateFromPlugins(plugins)
	m.setState(loader, registry, config)

	counts := registry.Count()
	slog.Info("Plugin system reloaded",
		"plugins_loaded", len(plugins),
		"tools", counts["tool"],
		"hooks", counts["hook"],
		"skills", counts["skill"],
		"mcp_servers", counts["mcp"],
		"collisions", len(registry.Collisions()),
	)

	return nil
}
