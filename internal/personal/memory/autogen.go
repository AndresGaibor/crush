package memory

import (
	"regexp"
	"strings"
	"sync"
)

// PatternDetector detecta patrones repetitivos en las interacciones
// del usuario que deberían convertirse en memorias.
type PatternDetector struct {
	// correcciones trackea correcciones que el usuario hace al agente
	// key: patrón normalizado, value: ocurrencias
	correcciones map[string]int
	mu           sync.Mutex
	minFrequency int
}

// NewPatternDetector crea un detector con la frecuencia mínima para sugerir.
func NewPatternDetector(minFrequency int) *PatternDetector {
	return &PatternDetector{
		correcciones: make(map[string]int),
		minFrequency: minFrequency,
	}
}

// Observe registra una observación del comportamiento del usuario.
// category: "correction" (usuario corrigió algo), "preference" (usuario indicó preferencia)
// pattern: el patrón observado (ej: "usa gin en vez de echo", "siempre usa tabs")
func (d *PatternDetector) Observe(category, pattern string) {
	if category == "" || pattern == "" {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	key := normalizePattern(category, pattern)
	d.correcciones[key]++
}

// CheckSuggestions retorna patrones que han alcanzado la frecuencia mínima
// y podrían sugerirse como nuevas memorias.
func (d *PatternDetector) CheckSuggestions() []SuggestedMemory {
	d.mu.Lock()
	defer d.mu.Unlock()

	var suggestions []SuggestedMemory
	for pattern, count := range d.correcciones {
		if count >= d.minFrequency {
			suggestions = append(suggestions, SuggestedMemory{
				Pattern:  pattern,
				Count:    count,
				Category: extractCategory(pattern),
			})
		}
	}
	return suggestions
}

// Reset limpia los conteos de patrones.
func (d *PatternDetector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.correcciones = make(map[string]int)
}

// SuggestedMemory representa una memoria sugerida por el detector de patrones.
type SuggestedMemory struct {
	Pattern  string
	Count    int
	Category string // "correction", "preference", "convention"
}

func normalizePattern(category, pattern string) string {
	return strings.ToLower(category + ":" + strings.TrimSpace(pattern))
}

func extractCategory(key string) string {
	if idx := strings.Index(key, ":"); idx >= 0 {
		return key[:idx]
	}
	return "unknown"
}

// GenerateMemoryContent genera el contenido Markdown para una memoria sugerida.
func GenerateMemoryContent(suggestion SuggestedMemory) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("tags: [")
	sb.WriteString(suggestion.Category)
	sb.WriteString(", auto-generated]\n")
	sb.WriteString("---\n\n")
	sb.WriteString("# ")
	sb.WriteString(capitalizeFirst(suggestion.Category))
	sb.WriteString("\n\n")
	sb.WriteString("Patrón detectado automáticamente (")
	sb.WriteString(strings.Repeat("*", suggestion.Count))
	sb.WriteString(" ocurrencias):\n\n")
	sb.WriteString(suggestion.Pattern)
	sb.WriteString("\n\n")
	sb.WriteString("## Instrucciones\n\n")
	sb.WriteString("Siempre que aplique, sigue este patrón sin preguntar.\n")

	return sb.String()
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// commonCorrections es una lista de patrones comunes a buscar en las
// correcciones del usuario para detectar automáticamente.
var commonCorrections = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:usa|prefiere|quiero|siempre)\s+(.+)`),
	regexp.MustCompile(`(?i)(?:no uses?|evita|nunca)\s+(.+)`),
	regexp.MustCompile(`(?i)(?:cambia|reemplaza|sustituye)\s+(.+)`),
	regexp.MustCompile(`(?i)(?:formato|estilo|convención)\s*:\s*(.+)`),
}

// ExtractCorrectionsFromDiff intenta extraer correcciones de un diff.
// Esto se puede llamar después de cada tool use de tipo "edit" o "write".
func ExtractCorrectionsFromDiff(diffOutput string) []string {
	var corrections []string
	for _, re := range commonCorrections {
		matches := re.FindAllStringSubmatch(diffOutput, -1)
		for _, match := range matches {
			if len(match) > 1 && strings.TrimSpace(match[1]) != "" {
				corrections = append(corrections, strings.TrimSpace(match[1]))
			}
		}
	}
	return corrections
}
