package plugins

import (
	"encoding/json"
	"time"
)

// PluginID es el identificador único de un plugin.
// Formato: "name@scope" (ej: "code-formatter@project")
type PluginID string

// PluginStatus representa el estado de un plugin.
type PluginStatus string

const (
	StatusLoaded   PluginStatus = "loaded"
	StatusEnabled  PluginStatus = "enabled"
	StatusDisabled PluginStatus = "disabled"
	StatusError    PluginStatus = "error"
)

// PluginScope indica de dónde viene el plugin.
type PluginScope string

const (
	ScopeProject PluginScope = "project" // .claude-plugin/ en el proyecto
	ScopeGlobal  PluginScope = "global"  // ~/.config/crush/plugins/
)

// Plugin es la representación completa de un plugin cargado.
type Plugin struct {
	ID          PluginID     `json:"id"`
	Name        string       `json:"name"`
	Version     string       `json:"version,omitempty"`
	Description string       `json:"description,omitempty"`
	Author      string       `json:"author,omitempty"`
	License     string       `json:"license,omitempty"`
	Keywords    []string     `json:"keywords,omitempty"`
	Scope       PluginScope  `json:"scope"`
	RootDir     string       `json:"-"` // Directorio raíz del plugin
	Manifest    *Manifest    `json:"-"`
	Status      PluginStatus `json:"status"`
	LoadedAt    time.Time    `json:"loaded_at"`
	Error       string       `json:"error,omitempty"`
}

// Manifest es el contenido parseado de plugin.json.
type Manifest struct {
	Name        string                        `json:"name"`
	Version     string                        `json:"version,omitempty"`
	Description string                        `json:"description,omitempty"`
	Author      string                        `json:"author,omitempty"`
	License     string                        `json:"license,omitempty"`
	Keywords    []string                      `json:"keywords,omitempty"`
	Tools       []ToolDecl                    `json:"tools,omitempty"`
	Skills      []string                      `json:"skills,omitempty"`
	Hooks       json.RawMessage               `json:"hooks,omitempty"`       // path string or inline hooks
	MCPServers  map[string]MCPDecl            `json:"mcpServers,omitempty"`
	Commands    []json.RawMessage             `json:"commands,omitempty"`
	Agents      []string                      `json:"agents,omitempty"`
	UserConfig  map[string]UserConfigField    `json:"userConfig,omitempty"`
	Settings    json.RawMessage               `json:"settings,omitempty"`
}

// ToolDecl declara una herramienta que el plugin proporciona.
type ToolDecl struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Source      string          `json:"source"`               // Path al .md con la tool
	InputSchema json.RawMessage `json:"inputSchema,omitempty"` // JSON Schema para input
}

// MCPDecl declara un servidor MCP que el plugin proporciona.
type MCPDecl struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Type    string            `json:"type,omitempty"` // "stdio" | "sse" | "http"
	URL     string            `json:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// UserConfigField define un campo de configuración del usuario para el plugin.
type UserConfigField struct {
	Type        string `json:"type"`                   // "string"|"number"|"boolean"
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Default     any    `json:"default,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"`
}

// PluginConfig es la configuración de plugins desde crush.json.
type PluginConfig struct {
	EnabledPlugins map[PluginID]PluginEnable     `json:"enabled_plugins,omitempty"`
	PluginOptions  map[PluginID]map[string]any   `json:"plugin_options,omitempty"`
	PluginsDir     string                        `json:"plugins_dir,omitempty"`
}

// PluginEnable puede ser true, false, o un array de versiones permitidas.
type PluginEnable any // bool | []string

// IsEnabled retorna true si el plugin está habilitado en la config.
func (c *PluginConfig) IsEnabled(id PluginID) bool {
	if c == nil || c.EnabledPlugins == nil {
		return true // Si no hay config, todo está habilitado por defecto
	}
	val, ok := c.EnabledPlugins[id]
	if !ok {
		return true // No especificado = habilitado
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return true
}

// GetOptions retorna las opciones de usuario para un plugin.
func (c *PluginConfig) GetOptions(id PluginID) map[string]any {
	if c == nil || c.PluginOptions == nil {
		return nil
	}
	return c.PluginOptions[id]
}

// RegistryEntry es una extensión registrada por un plugin.
type RegistryEntry struct {
	PluginID PluginID
	Type     string // "tool", "hook", "skill", "mcp", "command", "agent"
	Name     string
	Data     any
}
