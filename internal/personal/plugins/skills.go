package plugins

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// GetPluginSkillPaths retorna los paths de directorios de skills
// que los plugins proporcionan. Estos paths se agregan a la
// lista de skills paths del config para que el sistema de skills
// los descubra automáticamente.
func GetPluginSkillPaths(entries []RegistryEntry) []string {
	var paths []string
	for _, entry := range entries {
		if entry.Type != "skill" {
			continue
		}
		resolved, ok := entry.Data.(string)
		if !ok || resolved == "" {
			continue
		}

		// Verificar que el path existe y contiene SKILL.md
		if isValidSkillDir(resolved) {
			paths = append(paths, resolved)
		} else {
			slog.Warn("Plugin skill directory invalid or missing SKILL.md",
				"plugin", entry.PluginID,
				"path", resolved,
			)
		}
	}
	return paths
}

// isValidSkillDir verifica que un directorio contiene SKILL.md.
func isValidSkillDir(dir string) bool {
	// Buscar SKILL.md directamente o en subdirectorios
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() == "SKILL.md" {
			return true
		}
	}
	return false
}

// GetPluginSkillXML genera XML de skills para inyección en el prompt,
// similar a lo que hace skills.ToPromptXML() pero con prefijo del plugin.
func GetPluginSkillXML(entries []RegistryEntry) string {
	var xml strings.Builder
	xml.WriteString("<available_plugin_skills>\n")

	for _, entry := range entries {
		if entry.Type != "skill" {
			continue
		}
		resolved, ok := entry.Data.(string)
		if !ok {
			continue
		}

		// Leer el SKILL.md para obtener nombre y descripción
		skillPath := filepath.Join(resolved, "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			continue
		}

		fm, _ := splitFrontmatter(string(data))
		name, _ := fm["name"].(string)
		if name == "" {
			name = filepath.Base(resolved)
		}
		description, _ := fm["description"].(string)

		xml.WriteString("  <skill>\n")
		xml.WriteString("    <name>")
		xml.WriteString(entry.Name) // pluginName:skillDir
		xml.WriteString("</name>\n")
		if description != "" {
			xml.WriteString("    <description>")
			xml.WriteString(description)
			xml.WriteString("</description>\n")
		}
		xml.WriteString("    <location>")
		xml.WriteString(skillPath)
		xml.WriteString("</location>\n")
		xml.WriteString("  </skill>\n")
	}

	xml.WriteString("</available_plugin_skills>")
	return xml.String()
}
