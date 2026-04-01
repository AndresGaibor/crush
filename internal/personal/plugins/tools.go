package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"charm.land/fantasy"
)

// BuildPluginTools convierte las declaraciones de herramientas de plugins
// en fantasy.AgentTool que se pueden agregar al agente.
// Recibe las entries del Registry de tipo "tool".
func BuildPluginTools(entries []RegistryEntry, workingDir string) ([]fantasy.AgentTool, error) {
	var tools []fantasy.AgentTool

	for _, entry := range entries {
		if entry.Type != "tool" {
			continue
		}

		toolDecl, ok := entry.Data.(ToolDecl)
		if !ok {
			continue
		}

		// Leer el archivo .md de la herramienta
		toolMD, err := resolveAndReadToolMD(workingDir, entry.PluginID, toolDecl)
		if err != nil {
			slog.Warn("Failed to read plugin tool markdown",
				"plugin", entry.PluginID,
				"tool", toolDecl.Name,
				"error", err,
			)
			continue
		}

		// Parsear YAML frontmatter + contenido markdown
		frontmatter, instructions := splitFrontmatter(toolMD)

		// La descripción del tool es el contenido markdown completo
		// (frontmatter como metadata, body como instrucciones)
		description := buildToolDescription(toolDecl, frontmatter, instructions)

		// Crear el schema de input
		var inputSchema map[string]any
		if toolDecl.InputSchema != nil {
			json.Unmarshal(toolDecl.InputSchema, &inputSchema)
		} else if fmSchema := extractSchemaFromFrontmatter(frontmatter); fmSchema != nil {
			inputSchema = fmSchema
		}

		// Crear la herramienta usando fantasy
		tool := createPluginTool(toolDecl.Name, description, inputSchema, entry.PluginID)

		tools = append(tools, tool)
	}

	return tools, nil
}

// createPluginTool crea un fantasy.AgentTool a partir de la declaración del plugin.
func createPluginTool(name, description string, inputSchema map[string]any, pluginID PluginID) fantasy.AgentTool {
	// Nombre del tool con prefijo del plugin para evitar colisiones
	displayName := fmt.Sprintf("plugin_%s_%s", strings.ReplaceAll(string(pluginID), "@", "_"), name)

	return fantasy.NewAgentTool(
		displayName,
		description,
		func(ctx context.Context, params map[string]any, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			// Los plugins markdown son instrucciones para el agente,
			// no ejecutan código directamente. El agente lee las instrucciones
			// y las sigue usando las herramientas existentes.
			inputJSON, _ := json.Marshal(params)
			return fantasy.NewTextResponse(fmt.Sprintf(
				"Plugin tool '%s' invoked.\nInput: %s\n\nFollow the tool's instructions to complete this request.",
				name, string(inputJSON),
			)), nil
		},
	)
}

// resolveAndReadToolMD busca y lee el archivo markdown de una tool.
func resolveAndReadToolMD(workingDir string, pluginID PluginID, decl ToolDecl) (string, error) {
	// El Source es relativo al root del plugin
	// Buscar el plugin en el directorio de plugins
	pluginName := strings.Split(string(pluginID), "@")[0]

	searchDirs := []string{
		filepath.Join(workingDir, ".claude-plugin"),
		filepath.Join(workingDir, ".claude-plugin", "plugins", pluginName),
	}

	for _, dir := range searchDirs {
		path := filepath.Join(dir, decl.Source)
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}

	return "", fmt.Errorf("tool source file not found: %s", decl.Source)
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

	// Parsear YAML simple (solo strings y strings arrays por ahora)
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
			// Array simple
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

// extractSchemaFromFrontmatter intenta extraer input_schema del frontmatter.
func extractSchemaFromFrontmatter(fm map[string]any) map[string]any {
	if schema, ok := fm["input_schema"]; ok {
		if m, ok := schema.(map[string]any); ok {
			return m
		}
	}
	return nil
}

// buildToolDescription construye la descripción completa de la herramienta.
func buildToolDescription(decl ToolDecl, frontmatter map[string]any, instructions string) string {
	var sb strings.Builder

	// Metadata del plugin
	sb.WriteString(fmt.Sprintf("# %s\n\n", decl.Name))
	if decl.Description != "" {
		sb.WriteString(decl.Description)
		sb.WriteString("\n\n")
	}

	// Instrucciones del tool (body del .md)
	if instructions != "" {
		sb.WriteString(instructions)
	}

	return sb.String()
}
