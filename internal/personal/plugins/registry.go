package plugins

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
)

// Registry es el registro central de extensiones proporcionadas por plugins.
// Cuando un plugin se carga, registra sus tools, hooks, skills, etc. aquí.
// El coordinator consulta el Registry para obtener todas las extensiones activas.
type Registry struct {
	mu       sync.RWMutex
	entries  []RegistryEntry
	byType   map[string][]RegistryEntry // type → entries
	byPlugin map[PluginID][]RegistryEntry // plugin → entries
}

// NewRegistry crea un nuevo Registry vacío.
func NewRegistry() *Registry {
	return &Registry{
		byType:   make(map[string][]RegistryEntry),
		byPlugin: make(map[PluginID][]RegistryEntry),
	}
}

// Register agrega una extensión al registro.
func (r *Registry) Register(entry RegistryEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = append(r.entries, entry)
	r.byType[entry.Type] = append(r.byType[entry.Type], entry)
	r.byPlugin[entry.PluginID] = append(r.byPlugin[entry.PluginID], entry)

	slog.Debug("Plugin extension registered",
		"plugin", entry.PluginID,
		"type", entry.Type,
		"name", entry.Name,
	)
}

// Unregister elimina todas las extensiones de un plugin.
func (r *Registry) Unregister(pluginID PluginID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.byPlugin, pluginID)

	// Reconstruir entries y byType sin las del plugin
	var newEntries []RegistryEntry
	for _, e := range r.entries {
		if e.PluginID != pluginID {
			newEntries = append(newEntries, e)
		}
	}
	r.entries = newEntries

	r.byType = make(map[string][]RegistryEntry)
	for _, e := range r.entries {
		r.byType[e.Type] = append(r.byType[e.Type], e)
	}
}

// GetByType retorna todas las extensiones de un tipo dado.
func (r *Registry) GetByType(typ string) []RegistryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]RegistryEntry{}, r.byType[typ]...)
}

// GetTools retorna todas las herramientas registradas por plugins.
func (r *Registry) GetTools() []RegistryEntry {
	return r.GetByType("tool")
}

// GetHooks retorna todas las configuraciones de hooks de plugins.
func (r *Registry) GetHooks() []RegistryEntry {
	return r.GetByType("hook")
}

// GetSkills retorna los paths de skills de plugins.
func (r *Registry) GetSkills() []RegistryEntry {
	return r.GetByType("skill")
}

// GetMCPServers retorna los servidores MCP de plugins.
func (r *Registry) GetMCPServers() []RegistryEntry {
	return r.GetByType("mcp")
}

// All retorna todas las extensiones registradas.
func (r *Registry) All() []RegistryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]RegistryEntry{}, r.entries...)
}

// Count retorna la cantidad de extensiones por tipo.
func (r *Registry) Count() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	counts := make(map[string]int)
	for typ, entries := range r.byType {
		counts[typ] = len(entries)
	}
	return counts
}

// PluginCount retorna la cantidad de plugins con extensiones registradas.
func (r *Registry) PluginCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byPlugin)
}

// String retorna un resumen legible del registro.
func (r *Registry) String() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	counts := r.Count()
	var parts []string
	for typ, count := range counts {
		parts = append(parts, fmt.Sprintf("%s:%d", typ, count))
	}
	sort.Strings(parts)
	return fmt.Sprintf("Registry(%s)", strings.Join(parts, ", "))
}

// PopulateFromPlugins registra todas las extensiones de los plugins dados.
func (r *Registry) PopulateFromPlugins(plugins []*Plugin) {
	for _, p := range plugins {
		if p.Status != StatusEnabled || p.Manifest == nil {
			continue
		}

		// Registrar tools
		for _, tool := range p.Manifest.Tools {
			r.Register(RegistryEntry{
				PluginID: p.ID,
				Type:     "tool",
				Name:     tool.Name,
				Data:     tool,
			})
		}

		// Registrar skills (paths)
		for _, skillPath := range p.Manifest.Skills {
			resolved, err := ResolvePath(p.RootDir, skillPath)
			if err != nil {
				slog.Warn("Invalid skill path in plugin", "plugin", p.ID, "path", skillPath)
				continue
			}
			name := strings.TrimPrefix(resolved, p.RootDir+"/")
			r.Register(RegistryEntry{
				PluginID: p.ID,
				Type:     "skill",
				Name:     fmt.Sprintf("%s:%s", p.Name, name),
				Data:     resolved,
			})
		}

		// Registrar hooks (config JSON)
		if p.Manifest.Hooks != nil {
			r.Register(RegistryEntry{
				PluginID: p.ID,
				Type:     "hook",
				Name:     p.Name,
				Data:     p.Manifest.Hooks,
			})
		}

		// Registrar MCP servers
		for serverName, mcpDecl := range p.Manifest.MCPServers {
			r.Register(RegistryEntry{
				PluginID: p.ID,
				Type:     "mcp",
				Name:     fmt.Sprintf("plugin:%s:%s", p.Name, serverName),
				Data:     mcpDecl,
			})
		}
	}
}
