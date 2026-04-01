package memory

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/crush/internal/home"
)

// Scanner busca memorias relevantes para una consulta dada.
// Implementa búsqueda híbrida: primero por tags, luego por palabras clave en el contenido.
type Scanner struct {
	manager *MemoryManager
}

// NewScanner crea un nuevo Scanner.
func NewScanner(manager *MemoryManager) *Scanner {
	return &Scanner{manager: manager}
}

// FindRelevant busca memorias relevantes para la consulta dada.
// Retorna las memorias ordenadas por relevancia (más relevantes primero).
// maxResults limita la cantidad de resultados (0 = sin límite).
func (s *Scanner) FindRelevant(query string, maxResults int) ([]Memory, error) {
	memories, err := s.manager.All()
	if err != nil {
		return nil, err
	}

	if query == "" {
		return memories, nil
	}

	// Normalizar la consulta
	queryLower := strings.ToLower(query)
	queryTerms := strings.Fields(queryLower)

	// Calcular score para cada memoria
	type scored struct {
		mem   Memory
		score float64
	}

	var results []scored
	for _, mem := range memories {
		content, err := mem.GetContent()
		if err != nil {
			continue
		}

		var score float64

		// Score 1: Coincidencia de tags (peso alto)
		for _, tag := range mem.Tags {
			tagLower := strings.ToLower(tag)
			for _, term := range queryTerms {
				if strings.Contains(tagLower, term) {
					score += 10.0
				}
			}
		}

		// Score 2: Coincidencia en el nombre/ID (peso medio)
		idLower := strings.ToLower(mem.ID)
		for _, term := range queryTerms {
			if strings.Contains(idLower, term) {
				score += 5.0
			}
		}

		// Score 3: Coincidencia de términos en el contenido (peso bajo)
		contentLower := strings.ToLower(content)
		for _, term := range queryTerms {
			count := strings.Count(contentLower, term)
			if count > 0 {
				// Logarithmic scoring para evitar que archivos enormes dominen
				score += 1.0 + float64(min(count, 10))*0.5
			}
		}

		// Bonus: memorias de proyecto tienen prioridad sobre globales
		if mem.Scope == ScopeProject {
			score += 1.0
		}

		// Bonus: memorias más recientes tienen prioridad leve
		score += float64(mem.UpdatedAt.Unix()) / 1e12

		if score > 0 {
			results = append(results, scored{mem: mem, score: score})
		}
	}

	// Ordenar por score descendente
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Limitar resultados
	var filtered []Memory
	for i, r := range results {
		if maxResults > 0 && i >= maxResults {
			break
		}
		filtered = append(filtered, r.mem)
	}

	return filtered, nil
}

// FindByTag busca memorias que tengan un tag específico.
func (s *Scanner) FindByTag(tag string) ([]Memory, error) {
	memories, err := s.manager.All()
	if err != nil {
		return nil, err
	}

	tagLower := strings.ToLower(tag)
	var results []Memory
	for _, mem := range memories {
		for _, t := range mem.Tags {
			if strings.ToLower(t) == tagLower {
				results = append(results, mem)
				break
			}
		}
	}
	return results, nil
}

// FormatMemoryPaths retorna las rutas de las memorias relevantes
// en el formato que Crush ya usa para context_paths.
// Este es el punto de integración clave: los paths que retorna
// se inyectan en el system prompt automáticamente.
func (s *Scanner) FormatMemoryPaths(query string) ([]string, error) {
	relevant, err := s.FindRelevant(query, 5)
	if err != nil {
		return nil, err
	}

	// Siempre incluir CLAUDE.md si existe (prioridad máxima)
	var paths []string
	// Buscar CLAUDE.md en el directorio del proyecto
	for _, name := range []string{
		"CLAUDE.md",
		"crush.md",
		"CRUSH.md",
		"Crush.md",
	} {
		path := filepath.Join(s.manager.projectDir, name)
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		}
	}

	// Agregar memorias del proyecto relevantes
	for _, mem := range relevant {
		if mem.Scope == ScopeProject {
			paths = append(paths, mem.Path)
		}
	}

	// Agregar memorias globales relevantes
	for _, mem := range relevant {
		if mem.Scope == ScopeGlobal {
			paths = append(paths, mem.Path)
		}
	}

	return paths, nil
}

// ListProjectMemoryFiles retorna los archivos .md del directorio .crush/memory/
// como lista de paths relativos al project dir (para context_paths).
func (s *Scanner) ListProjectMemoryFiles() []string {
	entries, err := os.ReadDir(s.manager.projectMemDir)
	if err != nil {
		return nil
	}

	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			paths = append(paths, filepath.Join(".crush", "memory", entry.Name()))
		}
	}
	return paths
}

// ListGlobalMemoryFiles retorna los archivos .md del directorio global de memoria.
func (s *Scanner) ListGlobalMemoryFiles() []string {
	entries, err := os.ReadDir(s.manager.globalMemDir)
	if err != nil {
		return nil
	}

	homeDir := home.Dir()
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			// Retornar path absoluto para globales
			paths = append(paths, filepath.Join(homeDir, "memory", entry.Name()))
		}
	}
	return paths
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
