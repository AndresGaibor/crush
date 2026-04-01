package plugins

import (
	"log/slog"
)

// GetPluginMCPServers es un stub para Phase 2 de MCP integration.
// Por ahora retorna un mapa vacío.
// Cuando se implemente, convertirá declaraciones MCP de plugins
// al formato MCPConfig de Crush para registro automático.
func GetPluginMCPServers(entries []RegistryEntry) map[string]any {
	servers := make(map[string]any)

	for _, entry := range entries {
		if entry.Type != "mcp" {
			continue
		}

		mcpDecl, ok := entry.Data.(MCPDecl)
		if !ok {
			continue
		}

		slog.Debug("Plugin MCP server stub",
			"plugin", entry.PluginID,
			"server", entry.Name,
		)

		// Placeholder para integración futura
		servers[entry.Name] = mcpDecl
	}

	return servers
}
