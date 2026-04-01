package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"charm.land/fantasy"
)

// BuildPluginTools convierte las declaraciones de herramientas de plugins
// en fantasy.AgentTool que se pueden agregar al agente.
func BuildPluginTools(entries []RegistryEntry) ([]fantasy.AgentTool, error) {
	var tools []fantasy.AgentTool
	manager := GetManager()

	for _, entry := range entries {
		if entry.Type != "tool" {
			continue
		}

		toolDecl, ok := entry.Data.(ToolDecl)
		if !ok {
			continue
		}

		plugin := (*Plugin)(nil)
		if manager != nil && manager.Loader != nil {
			plugin = manager.Loader.GetPlugin(entry.PluginID)
		}
		if plugin == nil {
			slog.Warn("Skipping plugin tool because plugin could not be resolved",
				"plugin", entry.PluginID,
				"tool", toolDecl.Name,
			)
			continue
		}

		toolMD, err := resolveAndReadToolMD(plugin.RootDir, toolDecl)
		if err != nil {
			slog.Warn("Failed to read plugin tool markdown",
				"plugin", entry.PluginID,
				"tool", toolDecl.Name,
				"error", err,
			)
			continue
		}

		frontmatter, instructions := splitFrontmatter(toolMD)
		var options map[string]any
		if manager != nil && manager.Config != nil {
			options = manager.Config.GetOptions(entry.PluginID)
		}

		description := buildToolDescription(toolDecl, frontmatter, instructions, options)

		inputSchema := buildToolInputSchema(toolDecl, frontmatter)
		tools = append(tools, &pluginTool{
			name:        toolDecl.Name,
			description: description,
			schema:      inputSchema,
			pluginID:    entry.PluginID,
			toolPath:    toolSourcePath(plugin.RootDir, toolDecl.Source),
		})
	}

	return tools, nil
}

type pluginTool struct {
	name        string
	description string
	schema      map[string]any
	pluginID    PluginID
	toolPath    string
	provider    fantasy.ProviderOptions
}

func (t *pluginTool) Info() fantasy.ToolInfo {
	required := schemaRequiredFields(t.schema)
	parameters := t.schema
	if parameters == nil {
		parameters = map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	return fantasy.ToolInfo{
		Name:        t.name,
		Description: t.description,
		Parameters:  parameters,
		Required:    required,
		Parallel:    false,
	}
}

func (t *pluginTool) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
	inputJSON := strings.TrimSpace(params.Input)
	if inputJSON == "" {
		inputJSON = "{}"
	}

	return fantasy.NewTextResponse(fmt.Sprintf(
		"Plugin tool %q is a Markdown-defined instruction tool.\nPlugin: %s\nSource: %s\nInput: %s\n\nFollow the tool instructions in the plugin manifest to complete the request.",
		t.name, t.pluginID, t.toolPath, inputJSON,
	)), nil
}

func (t *pluginTool) ProviderOptions() fantasy.ProviderOptions {
	return t.provider
}

func (t *pluginTool) SetProviderOptions(opts fantasy.ProviderOptions) {
	t.provider = opts
}

// resolveAndReadToolMD busca y lee el archivo markdown de una tool.
func resolveAndReadToolMD(rootDir string, decl ToolDecl) (string, error) {
	path, err := toolSourcePathResolved(rootDir, decl.Source)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func toolSourcePath(rootDir, source string) string {
	path, err := toolSourcePathResolved(rootDir, source)
	if err != nil {
		return ""
	}
	return path
}

func toolSourcePathResolved(rootDir, source string) (string, error) {
	if rootDir == "" {
		return "", fmt.Errorf("plugin root directory is empty")
	}

	resolved, err := ResolvePath(rootDir, source)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("tool source %q points to a directory", source)
	}
	return resolved, nil
}

// splitFrontmatter separa el YAML frontmatter del contenido markdown.
func splitFrontmatter(content string) (frontmatter map[string]any, body string) {
	frontmatter = make(map[string]any)
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "---") {
		return frontmatter, content
	}

	var endIdx int
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx == 0 {
		return frontmatter, content
	}

	// Parsear YAML simple (solo strings y strings arrays por ahora).
	fmText := strings.Join(lines[1:endIdx], "\n")
	parseSimpleYAML(fmText, frontmatter)

	body = strings.Join(lines[endIdx+1:], "\n")
	return frontmatter, body
}

// parseSimpleYAML parsea YAML simple (solo strings y arrays de strings).
func parseSimpleYAML(text string, m map[string]any) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "[") {
			value = strings.Trim(value, "[]")
			var items []string
			for _, item := range strings.Split(value, ",") {
				item = strings.TrimSpace(item)
				item = strings.Trim(item, "\"'")
				if item != "" {
					items = append(items, item)
				}
			}
			m[key] = items
		} else {
			value = strings.Trim(value, "\"'")
			m[key] = value
		}
	}
}

// buildToolInputSchema construye el schema de entrada del tool.
func buildToolInputSchema(decl ToolDecl, frontmatter map[string]any) map[string]any {
	var schema map[string]any
	if err := json.Unmarshal(decl.InputSchema, &schema); err == nil && len(schema) > 0 {
		return schema
	}

	if fmSchema := extractSchemaFromFrontmatter(frontmatter); fmSchema != nil {
		return fmSchema
	}

	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// extractSchemaFromFrontmatter intenta extraer input_schema del frontmatter.
func extractSchemaFromFrontmatter(fm map[string]any) map[string]any {
	if schema, ok := fm["input_schema"]; ok {
		if m, ok := schema.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func schemaRequiredFields(schema map[string]any) []string {
	if schema == nil {
		return nil
	}
	required, ok := schema["required"]
	if !ok {
		return nil
	}

	switch value := required.(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// buildToolDescription construye la descripción completa de la herramienta.
func buildToolDescription(decl ToolDecl, frontmatter map[string]any, instructions string, options map[string]any) string {
	var sb strings.Builder

	// Metadata del plugin.
	sb.WriteString(fmt.Sprintf("# %s\n\n", decl.Name))
	if decl.Description != "" {
		sb.WriteString(decl.Description)
		sb.WriteString("\n\n")
	} else if fmDescription, ok := frontmatter["description"].(string); ok && fmDescription != "" {
		sb.WriteString(fmDescription)
		sb.WriteString("\n\n")
	}

	// Instrucciones del tool (body del .md).
	if instructions != "" {
		sb.WriteString(instructions)
		sb.WriteString("\n\n")
	}

	if len(options) > 0 {
		sb.WriteString("## Plugin configuration\n\n")
		if data, err := json.MarshalIndent(options, "", "  "); err == nil {
			sb.Write(data)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
