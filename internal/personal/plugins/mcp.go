package plugins

import (
	"log/slog"

	appconfig "github.com/charmbracelet/crush/internal/config"
)

// GetPluginMCPServers convierte las declaraciones MCP de plugins al formato
// MCPConfig de Crush para registro automático.
func GetPluginMCPServers(entries []RegistryEntry) map[string]appconfig.MCPConfig {
	servers := make(map[string]appconfig.MCPConfig)

	for _, entry := range entries {
		if entry.Type != "mcp" {
			continue
		}

		mcpDecl, ok := entry.Data.(MCPDecl)
		if !ok {
			continue
		}

		mcpType := appconfig.MCPStdio
		switch mcpDecl.Type {
		case "":
			// Mantener stdio como default.
		case "stdio":
			mcpType = appconfig.MCPStdio
		case "sse":
			mcpType = appconfig.MCPSSE
		case "http":
			mcpType = appconfig.MCPHttp
		default:
			slog.Warn("Unsupported plugin MCP type, skipping",
				"plugin", entry.PluginID,
				"server", entry.Name,
				"type", mcpDecl.Type,
			)
			continue
		}

		servers[entry.Name] = appconfig.MCPConfig{
			Command: mcpDecl.Command,
			Args:    append([]string(nil), mcpDecl.Args...),
			Type:    mcpType,
			URL:     mcpDecl.URL,
			Env:     cloneStringMap(mcpDecl.Env),
			Timeout: mcpDecl.Timeout,
			Headers: cloneStringMap(mcpDecl.Headers),
		}

		slog.Info("Plugin MCP server registered",
			"plugin", entry.PluginID,
			"server", entry.Name,
			"type", mcpType,
		)
	}

	return servers
}

// MergePluginMCPIntoConfig combina los servidores MCP de plugins con la
// configuración existente, preservando los servidores definidos por el usuario.
func MergePluginMCPIntoConfig(
	existing map[string]appconfig.MCPConfig,
	pluginServers map[string]appconfig.MCPConfig,
) map[string]appconfig.MCPConfig {
	merged := make(map[string]appconfig.MCPConfig, len(existing)+len(pluginServers))
	for k, v := range existing {
		merged[k] = cloneMCPConfig(v)
	}
	for k, v := range pluginServers {
		if _, exists := merged[k]; exists {
			slog.Warn("Plugin MCP server name conflicts with user config, skipping", "server", k)
			continue
		}
		merged[k] = cloneMCPConfig(v)
	}
	return merged
}

func cloneMCPConfig(src appconfig.MCPConfig) appconfig.MCPConfig {
	dst := src
	dst.Args = append([]string(nil), src.Args...)
	dst.Env = cloneStringMap(src.Env)
	dst.Headers = cloneStringMap(src.Headers)
	dst.DisabledTools = append([]string(nil), src.DisabledTools...)
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
