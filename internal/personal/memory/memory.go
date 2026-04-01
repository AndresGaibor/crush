package memory

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/crush/internal/home"
	"gopkg.in/yaml.v3"
)

// Memory representa una memoria persistente almacenada como archivo Markdown.
type Memory struct {
	// ID es el nombre del archivo sin extensión (ej: "preferences", "patterns")
	ID string `json:"id"`
	// Path es la ruta absoluta al archivo Markdown
	Path string `json:"path"`
	// Scope indica si es "project" (en el repo) o "global" (en ~/.config/crush/memory/)
	Scope MemoryScope `json:"scope"`
	// Tags son etiquetas para búsqueda (extraídas del frontmatter o del contenido)
	Tags []string `json:"tags"`
	// CreatedAt es la fecha de creación
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt es la fecha de última modificación
	UpdatedAt time.Time `json:"updated_at"`
	// Size es el tamaño del archivo en bytes
	Size int64 `json:"size"`
	// Content es el contenido del archivo (cargado bajo demanda)
	Content string `json:"-"`
}

// MemoryScope indica el ámbito de una memoria.
type MemoryScope string

const (
	ScopeProject MemoryScope = "project"
	ScopeGlobal  MemoryScope = "global"
)

// MemoryManager gestiona el ciclo de vida de las memorias.
type MemoryManager struct {
	projectDir    string // Directorio del proyecto (working directory)
	projectMemDir string // .crush/memory/ dentro del proyecto
	globalMemDir  string // ~/.config/crush/memory/
}

// NewMemoryManager crea un nuevo MemoryManager.
func NewMemoryManager(projectDir string) (*MemoryManager, error) {
	projectMemDir := filepath.Join(projectDir, ".crush", "memory")
	globalMemDir := filepath.Join(home.Dir(), ".config", "crush", "memory")

	// Crear directorios si no existen
	for _, dir := range []string{projectMemDir, globalMemDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating memory directory %s: %w", dir, err)
		}
	}

	return &MemoryManager{
		projectDir:    projectDir,
		projectMemDir: projectMemDir,
		globalMemDir:  globalMemDir,
	}, nil
}

// All retorna todas las memorias (proyecto + global), sin cargar el contenido.
func (m *MemoryManager) All() ([]Memory, error) {
	var all []Memory
	for _, dir := range []struct {
		path  string
		scope MemoryScope
	}{
		{m.projectMemDir, ScopeProject},
		{m.globalMemDir, ScopeGlobal},
	} {
		memories, err := m.scanDir(dir.path, dir.scope)
		if err != nil {
			slog.Warn("Failed to scan memory directory", "dir", dir.path, "error", err)
			continue
		}
		all = append(all, memories...)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})
	return all, nil
}

// Project retorna solo las memorias del proyecto actual.
func (m *MemoryManager) Project() ([]Memory, error) {
	return m.scanDir(m.projectMemDir, ScopeProject)
}

// Global retorna solo las memorias globales del usuario.
func (m *MemoryManager) Global() ([]Memory, error) {
	return m.scanDir(m.globalMemDir, ScopeGlobal)
}

// Save guarda una memoria. Si ya existe la sobrescribe.
// El id se usa como nombre de archivo (ej: "preferences" → "preferences.md").
func (m *MemoryManager) Save(id string, content string, scope MemoryScope, tags []string) (*Memory, error) {
	if id == "" {
		return nil, fmt.Errorf("memory id cannot be empty")
	}
	// Sanitizar ID: solo letras, números, guiones, guiones bajos
	safeID := sanitizeFilename(id)

	var dir string
	switch scope {
	case ScopeGlobal:
		dir = m.globalMemDir
	default:
		dir = m.projectMemDir
	}

	// Si hay tags, agregar frontmatter YAML
	var finalContent string
	if len(tags) > 0 {
		var sb strings.Builder
		sb.WriteString("---\n")
		sb.WriteString("tags: [")
		for i, tag := range tags {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(strconv.Quote(tag))
		}
		sb.WriteString("]\n")
		sb.WriteString("---\n\n")
		sb.WriteString(content)
		finalContent = sb.String()
	} else {
		finalContent = content
	}

	path := filepath.Join(dir, safeID+".md")
	if err := os.WriteFile(path, []byte(finalContent), 0o644); err != nil {
		return nil, fmt.Errorf("writing memory file: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stating memory file: %w", err)
	}

	mem := &Memory{
		ID:        safeID,
		Path:      path,
		Scope:     scope,
		Tags:      append([]string(nil), tags...),
		CreatedAt: info.ModTime(), // Si sobrescribe, usa la fecha actual
		UpdatedAt: info.ModTime(),
		Size:      info.Size(),
		Content:   finalContent,
	}

	slog.Info("Memory saved", "id", safeID, "scope", scope, "size", info.Size())
	return mem, nil
}

// Load carga el contenido de una memoria específica.
func (m *MemoryManager) Load(id string) (*Memory, error) {
	// Buscar en proyecto primero, luego en global
	for _, dir := range []string{m.projectMemDir, m.globalMemDir} {
		path := filepath.Join(dir, sanitizeFilename(id)+".md")
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		info, _ := os.Stat(path)
		var scope MemoryScope
		if strings.HasPrefix(dir, m.globalMemDir) {
			scope = ScopeGlobal
		}
		return &Memory{
			ID:        sanitizeFilename(id),
			Path:      path,
			Scope:     scope,
			Tags:      extractTags(path),
			CreatedAt: info.ModTime(),
			Content:   string(content),
			UpdatedAt: info.ModTime(),
			Size:      info.Size(),
		}, nil
	}
	return nil, fmt.Errorf("memory not found: %s", id)
}

// Delete elimina una memoria por ID.
func (m *MemoryManager) Delete(id string) error {
	for _, dir := range []string{m.projectMemDir, m.globalMemDir} {
		path := filepath.Join(dir, sanitizeFilename(id)+".md")
		if err := os.Remove(path); err == nil {
			slog.Info("Memory deleted", "id", id)
			return nil
		}
	}
	return fmt.Errorf("memory not found: %s", id)
}

// GetContent carga el contenido de una memoria.
func (mem *Memory) GetContent() (string, error) {
	if mem.Content != "" {
		return mem.Content, nil
	}
	content, err := os.ReadFile(mem.Path)
	if err != nil {
		return "", fmt.Errorf("reading memory file: %w", err)
	}
	mem.Content = string(content)
	return mem.Content, nil
}

// ToContextFile convierte la memoria al formato ContextFile que Crush ya usa.
func (mem *Memory) ToContextFile() (*ContextFile, error) {
	content, err := mem.GetContent()
	if err != nil {
		return nil, err
	}
	return &ContextFile{
		Path:    mem.Path,
		Content: content,
	}, nil
}

// ContextFile es un alias del tipo de Crush para compatibilidad.
// Lo definimos aquí para que el módulo sea auto-contenido.
type ContextFile struct {
	Path    string
	Content string
}

// scanDir escanea un directorio buscando archivos .md de memoria.
func (m *MemoryManager) scanDir(dir string, scope MemoryScope) ([]Memory, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var memories []Memory
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".md")
		mem := Memory{
			ID:        id,
			Path:      filepath.Join(dir, entry.Name()),
			Scope:     scope,
			CreatedAt: info.ModTime(),
			UpdatedAt: info.ModTime(),
			Size:      info.Size(),
		}

		// Extraer tags del frontmatter YAML si existe
		mem.Tags = extractTags(mem.Path)

		memories = append(memories, mem)
	}
	return memories, nil
}

// sanitizeFilename limpia un ID para que sea seguro como nombre de archivo.
func sanitizeFilename(id string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(id) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	return sb.String()
}

// extractTags intenta extraer tags del frontmatter YAML de un archivo Markdown.
// Formato esperado: las primeras líneas con --- delimitado, contiene "tags: [a, b, c]"
func extractTags(path string) []string {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	frontmatter, ok := extractFrontmatter(string(content))
	if !ok {
		return nil
	}

	tags, err := parseTags(frontmatter)
	if err != nil {
		return nil
	}
	return tags
}

func extractFrontmatter(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", false
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), true
		}
	}
	return "", false
}

func parseTags(frontmatter string) ([]string, error) {
	var meta map[string]any
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return nil, err
	}

	rawTags, ok := meta["tags"]
	if !ok {
		return nil, nil
	}

	switch v := rawTags.(type) {
	case []any:
		tags := make([]string, 0, len(v))
		for _, item := range v {
			tag := strings.TrimSpace(fmt.Sprint(item))
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		return tags, nil
	case []string:
		tags := make([]string, 0, len(v))
		for _, tag := range v {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		return tags, nil
	default:
		tag := strings.TrimSpace(fmt.Sprint(v))
		if tag == "" {
			return nil, nil
		}
		return []string{tag}, nil
	}
}

// Recent retorna las memorias más recientes, limitadas por limit.
func (m *MemoryManager) Recent(limit int) ([]Memory, error) {
	memories, err := m.All()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || len(memories) <= limit {
		return memories, nil
	}
	return append([]Memory(nil), memories[:limit]...), nil
}

// ToJSON serializa una lista de memorias a JSON.
func ToJSON(memories []Memory) string {
	data, err := json.MarshalIndent(memories, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(data)
}
