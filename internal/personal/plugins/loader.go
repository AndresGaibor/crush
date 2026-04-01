package plugins

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Loader descubre y carga plugins desde directorios del filesystem.
type Loader struct {
	projectDir  string
	globalDir   string
	config      *PluginConfig
	mu          sync.RWMutex
	plugins     map[PluginID]*Plugin
	loadErrors  map[PluginID]error
}

// NewLoader crea un nuevo Loader de plugins.
func NewLoader(projectDir string, config *PluginConfig) *Loader {
	globalDir := filepath.Join(homeDir(), ".config", "crush", "plugins")
	return &Loader{
		projectDir: projectDir,
		globalDir:  globalDir,
		config:     config,
		plugins:    make(map[PluginID]*Plugin),
		loadErrors: make(map[PluginID]error),
	}
}

// LoadAll descubre y carga todos los plugins de proyecto y global.
func (l *Loader) LoadAll() ([]*Plugin, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var all []*Plugin

	// 1. Plugins de proyecto (mayor prioridad)
	projectPlugins, err := l.discoverPlugins(l.projectDir, ScopeProject)
	if err != nil {
		slog.Warn("Failed to discover project plugins", "error", err)
	}

	// 2. Plugins globales
	globalPlugins, err := l.discoverPlugins(l.globalDir, ScopeGlobal)
	if err != nil {
		slog.Warn("Failed to discover global plugins", "error", err)
	}

	// 3. Plugins desde config (rutas explícitas)
	configPlugins := l.discoverFromConfig()

	// Merge: proyecto > config > global (proyecto tiene prioridad)
	seen := make(map[string]bool)
	for _, p := range append(append(globalPlugins, configPlugins...), projectPlugins...) {
		if seen[p.Name] {
			continue // Ya cargado desde una fuente de mayor prioridad
		}
		seen[p.Name] = true

		if !l.config.IsEnabled(p.ID) {
			p.Status = StatusDisabled
			slog.Debug("Plugin disabled by config", "id", p.ID)
		} else {
			// Cargar manifest completo
			if err := l.loadPluginManifest(p); err != nil {
				p.Status = StatusError
				p.Error = err.Error()
				l.loadErrors[p.ID] = err
				slog.Warn("Failed to load plugin manifest",
					"id", p.ID, "error", err)
			} else {
				p.Status = StatusEnabled
			}
		}

		l.plugins[p.ID] = p
		all = append(all, p)
	}

	// Ordenar por nombre
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	slog.Info("Plugins loaded",
		"total", len(all),
		"enabled", len(all)-len(l.loadErrors),
		"errors", len(l.loadErrors),
	)

	return all, nil
}

// discoverPlugins escanea un directorio buscando subdirectorios con plugin.json.
func (l *Loader) discoverPlugins(baseDir string, scope PluginScope) ([]*Plugin, error) {
	var plugins []*Plugin

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(baseDir, entry.Name())

		// Buscar plugin.json en este directorio
		manifestPath, err := FindManifest(pluginDir)
		if err != nil {
			continue // No es un plugin válido
		}

		// Parseo rápido para obtener el nombre
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var quick struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &quick); err != nil || quick.Name == "" {
			continue
		}

		pluginID := PluginID(quick.Name + "@" + string(scope))
		plugin := &Plugin{
			ID:      pluginID,
			Name:    quick.Name,
			Scope:   scope,
			RootDir: pluginDir,
			Status:  StatusLoaded,
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// discoverFromConfig busca plugins en rutas explícitas de la config.
func (l *Loader) discoverFromConfig() []*Plugin {
	if l.config == nil || l.config.PluginsDir == "" {
		return nil
	}

	// Soportar múltiples directorios separados por ":"
	dirs := strings.Split(l.config.PluginsDir, ":")
	var plugins []*Plugin

	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		found, err := l.discoverPlugins(dir, ScopeGlobal)
		if err != nil {
			slog.Warn("Failed to discover plugins from config dir", "dir", dir, "error", err)
			continue
		}
		plugins = append(plugins, found...)
	}

	return plugins
}

// loadPluginManifest carga el manifest completo de un plugin.
func (l *Loader) loadPluginManifest(p *Plugin) error {
	manifestPath, err := FindManifest(p.RootDir)
	if err != nil {
		return fmt.Errorf("finding manifest: %w", err)
	}

	manifest, err := ParseManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	p.Manifest = manifest
	p.Version = manifest.Version
	p.Description = manifest.Description
	p.Author = manifest.Author
	p.License = manifest.License
	p.Keywords = manifest.Keywords

	return nil
}

// GetPlugin retorna un plugin por ID.
func (l *Loader) GetPlugin(id PluginID) *Plugin {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.plugins[id]
}

// GetAll retorna todos los plugins cargados.
func (l *Loader) GetAll() []*Plugin {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var all []*Plugin
	for _, p := range l.plugins {
		all = append(all, p)
	}
	return all
}

// GetEnabled retorna solo los plugins habilitados sin errores.
func (l *Loader) GetEnabled() []*Plugin {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var enabled []*Plugin
	for _, p := range l.plugins {
		if p.Status == StatusEnabled {
			enabled = append(enabled, p)
		}
	}
	return enabled
}

// homeDir retorna el directorio home del usuario.
func homeDir() string {
	dir, _ := os.UserHomeDir()
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "crush")
	}
	return dir
}
