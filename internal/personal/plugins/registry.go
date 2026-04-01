package plugins

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// RegistryCollision describe una colisión entre dos extensiones de plugins.
type RegistryCollision struct {
	Type           string
	Name           string
	WinnerPluginID PluginID
	LoserPluginID  PluginID
	Reason         string
}

// Registry es el registro central de extensiones proporcionadas por plugins.
// Cuando un plugin se carga, registra sus tools, hooks, skills, etc. aquí.
// El coordinator consulta el Registry para obtener todas las extensiones activas.
type Registry struct {
	mu         sync.RWMutex
	entries    []RegistryEntry
	byType     map[string][]RegistryEntry   // type → entries
	byPlugin   map[PluginID][]RegistryEntry // plugin → entries
	collisions []RegistryCollision
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

	var collisions []RegistryCollision
	for _, collision := range r.collisions {
		if collision.WinnerPluginID == pluginID || collision.LoserPluginID == pluginID {
			continue
		}
		collisions = append(collisions, collision)
	}
	r.collisions = collisions
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

// Collisions retorna las colisiones detectadas al registrar extensiones.
func (r *Registry) Collisions() []RegistryCollision {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]RegistryCollision{}, r.collisions...)
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
	seenTools := make(map[string]PluginID)
	seenSkills := make(map[string]PluginID)
	seenMCP := make(map[string]PluginID)

	for _, p := range plugins {
		if p.Status != StatusEnabled || p.Manifest == nil {
			continue
		}

		// Registrar tools
		for _, tool := range p.Manifest.Tools {
			if owner, ok := seenTools[tool.Name]; ok {
				r.recordCollision("tool", tool.Name, owner, p.ID, "duplicate tool name")
				continue
			}
			seenTools[tool.Name] = p.ID
			r.Register(RegistryEntry{
				PluginID: p.ID,
				Type:     "tool",
				Name:     fmt.Sprintf("%s:%s", p.ID, tool.Name),
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
			entryName := fmt.Sprintf("%s:%s", p.ID, filepath.Base(resolved))
			if owner, ok := seenSkills[entryName]; ok {
				r.recordCollision("skill", entryName, owner, p.ID, "duplicate skill path")
				continue
			}
			seenSkills[entryName] = p.ID
			r.Register(RegistryEntry{
				PluginID: p.ID,
				Type:     "skill",
				Name:     entryName,
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
			entryName := fmt.Sprintf("plugin:%s:%s", p.ID, serverName)
			if owner, ok := seenMCP[entryName]; ok {
				r.recordCollision("mcp", entryName, owner, p.ID, "duplicate MCP server name")
				continue
			}
			seenMCP[entryName] = p.ID
			r.Register(RegistryEntry{
				PluginID: p.ID,
				Type:     "mcp",
				Name:     entryName,
				Data:     mcpDecl,
			})
		}
	}
}

func (r *Registry) recordCollision(typ, name string, winner, loser PluginID, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.collisions = append(r.collisions, RegistryCollision{
		Type:           typ,
		Name:           name,
		WinnerPluginID: winner,
		LoserPluginID:  loser,
		Reason:         reason,
	})

	slog.Warn("Plugin extension collision detected",
		"type", typ,
		"name", name,
		"winner", winner,
		"loser", loser,
		"reason", reason,
	)
}
