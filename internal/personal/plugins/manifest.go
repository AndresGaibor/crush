package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// pluginNameRegex valida nombres de plugins: kebab-case, 2-64 chars.
	pluginNameRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// semverRegex valida versiones semver simples.
	semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?$`)
)

// manifestFileNames son los nombres de archivo que se buscan como manifiesto.
var manifestFileNames = []string{
	"plugin.json",
	".claude-plugin/plugin.json",
}

// FindManifest busca plugin.json en un directorio y subdirectorios.
// Retorna el path al manifiesto encontrado o error.
func FindManifest(dir string) (string, error) {
	for _, name := range manifestFileNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	// Buscar recursivamente un nivel
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading plugin directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			for _, name := range manifestFileNames {
				path := filepath.Join(dir, entry.Name(), name)
				if _, err := os.Stat(path); err == nil {
					return path, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no plugin.json found in %s", dir)
}

// ParseManifest lee y valida un archivo plugin.json.
func ParseManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest JSON: %w", err)
	}

	if err := ValidateManifest(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// ValidateManifest valida que un manifest tenga los campos obligatorios
// y que los valores sean correctos.
func ValidateManifest(m *Manifest) error {
	// Name (obligatorio)
	if m.Name == "" {
		return fmt.Errorf("plugin manifest: 'name' is required")
	}
	if len(m.Name) > 64 {
		return fmt.Errorf("plugin manifest: 'name' must be <= 64 characters")
	}
	if !pluginNameRegex.MatchString(m.Name) {
		return fmt.Errorf("plugin manifest: 'name' must be kebab-case (e.g., 'my-plugin'), got %q", m.Name)
	}

	// Version (opcional pero si existe, debe ser semver)
	if m.Version != "" && !semverRegex.MatchString(m.Version) {
		return fmt.Errorf("plugin manifest: 'version' must be semver (e.g., '1.0.0'), got %q", m.Version)
	}

	// Description (recomendado)
	if m.Description == "" {
		return fmt.Errorf("plugin manifest: 'description' is recommended")
	}

	// Validar tools
	for i, tool := range m.Tools {
		if tool.Name == "" {
			return fmt.Errorf("plugin manifest: tools[%d].name is required", i)
		}
		if tool.Source == "" {
			return fmt.Errorf("plugin manifest: tools[%d].source is required for tool %q", i, tool.Name)
		}
		// El nombre debe ser PascalCase
		if !isValidToolName(tool.Name) {
			return fmt.Errorf("plugin manifest: tools[%d].name %q must be PascalCase", i, tool.Name)
		}
	}

	// Validar skills (solo paths)
	for i, skill := range m.Skills {
		if skill == "" {
			return fmt.Errorf("plugin manifest: skills[%d] path is empty", i)
		}
	}

	return nil
}

// ResolvePath resuelve un path relativo al directorio raíz del plugin.
func ResolvePath(rootDir, relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return relPath, nil
	}
	// Prevenir path traversal en el path de entrada
	if strings.Contains(relPath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", relPath)
	}
	resolved := filepath.Join(rootDir, relPath)
	// También verificar que el resultado no sale del rootDir
	absRoot, _ := filepath.Abs(rootDir)
	absResolved, _ := filepath.Abs(resolved)
	if !strings.HasPrefix(absResolved, absRoot) {
		return "", fmt.Errorf("path traversal detected: %s", relPath)
	}
	return resolved, nil
}

// isValidToolName valida que el nombre de una herramienta sea válido.
func isValidToolName(name string) bool {
	if name == "" {
		return false
	}
	// PascalCase: primera letra mayúscula, luego letras/números
	matched := regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`).MatchString(name)
	return matched
}
