package plugins

import (
	"log/slog"
	"sync"
)

var (
	instance     *Manager
	instanceOnce sync.Once
)

// Manager es el gestor central del sistema de plugins.
// Coordina la carga, el registro y la integración con Crush.
type Manager struct {
	Loader   *Loader
	Registry *Registry
	Config   *PluginConfig
}

// Init inicializa el sistema de plugins completo.
// 1. Carga la configuración
// 2. Descubre y carga plugins
// 3. Puebla el registro con todas las extensiones
func Init(projectDir string) (*Manager, error) {
	var initErr error
	instanceOnce.Do(func() {
		// 1. Cargar config
		config, err := LoadConfig(projectDir)
		if err != nil {
			slog.Warn("Failed to load plugin config, using defaults", "error", err)
			config = &PluginConfig{}
		}

		// 2. Crear loader y descubrir plugins
		loader := NewLoader(projectDir, config)
		plugins, err := loader.LoadAll()
		if err != nil {
			initErr = err
			return
		}

		// 3. Crear registro y registrar extensiones
		registry := NewRegistry()
		registry.PopulateFromPlugins(plugins)

		instance = &Manager{
			Loader:   loader,
			Registry: registry,
			Config:   config,
		}

		counts := registry.Count()
		slog.Info("Plugin system initialized",
			"plugins_loaded", len(plugins),
			"tools", counts["tool"],
			"hooks", counts["hook"],
			"skills", counts["skill"],
			"mcp_servers", counts["mcp"],
		)
	})
	return instance, initErr
}

// GetManager retorna el Manager singleton.
func GetManager() *Manager {
	return instance
}

// IsInitialized retorna true si el sistema fue inicializado.
func IsInitialized() bool {
	return instance != nil
}

// Reset reinicia el singleton (para tests).
func Reset() {
	instanceOnce = sync.Once{}
	instance = nil
}
