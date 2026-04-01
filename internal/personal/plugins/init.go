package plugins

import (
	"fmt"
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
	mu         sync.RWMutex
	Loader     *Loader
	Registry   *Registry
	Config     *PluginConfig
	projectDir string
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
			Loader:     loader,
			Registry:   registry,
			Config:     config,
			projectDir: projectDir,
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

// setState reemplaza el estado interno del manager de forma atómica.
func (m *Manager) setState(loader *Loader, registry *Registry, config *PluginConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Loader = loader
	m.Registry = registry
	m.Config = config
}

// snapshot retorna una copia segura del estado actual del manager.
func (m *Manager) snapshot() (*Loader, *Registry, *PluginConfig, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.Loader, m.Registry, m.Config, m.projectDir
}

func (m *Manager) ensureProjectDir() error {
	if m == nil {
		return fmt.Errorf("plugin manager is nil")
	}
	_, _, _, projectDir := m.snapshot()
	if projectDir == "" {
		return fmt.Errorf("plugin manager project directory is empty")
	}
	return nil
}
