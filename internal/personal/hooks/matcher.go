package hooks

import (
	"path/filepath"
	"regexp"
	"strings"
)

// MatchHook evalúa si un hook con el matcher dado debe ejecutarse
// para el evento proporcionado.
//
// Estrategias de matching:
//  1. Vacío o "*" → matchea todo
//  2. Pipe-separated: "Write|Edit|Bash" → matchea exactamente contra tool_name
//  3. Glob con paréntesis: "Bash(rm *)" → glob match contra tool_name + input
//  4. Cualquier otro string → se interpreta como regex
func MatchHook(matcher string, event HookEvent) bool {
	if matcher == "" || matcher == "*" {
		return true
	}

	// Para eventos que no son de herramienta, el matcher aplica sobre el Source
	query := event.ToolName
	if event.Type == SessionStart {
		query = event.Source
	}

	// Estrategia 1: Pipe-separated exact matches
	if isSimpleMatcher(matcher) {
		parts := strings.Split(matcher, "|")
		for _, p := range parts {
			if strings.TrimSpace(p) == query {
				return true
			}
		}
		return false
	}

	// Estrategia 2: Glob con paréntesis "ToolName(pattern)"
	if idx := strings.Index(matcher, "("); idx > 0 {
		toolPart := matcher[:idx]
		globPart := strings.TrimSuffix(matcher[idx+1:], ")")

		// Verificar que el nombre de herramienta coincide
		if toolPart != query {
			return false
		}

		// Si el evento tiene input de tipo string, hacer glob match
		if inputStr, ok := event.ToolInput.(string); ok {
			matched, _ := filepath.Match(globPart, inputStr)
			return matched
		}
		// Si no hay input string, matchear si el glob está vacío o es "*"
		return globPart == "" || globPart == "*"
	}

	// Estrategia 3: Regex
	re, err := regexp.Compile(matcher)
	if err != nil {
		// Si no es regex válido, tratar como match exacto
		return matcher == query
	}
	return re.MatchString(query)
}

// isSimpleMatcher retorna true si el matcher contiene solo letras,
// números, pipes, underscores y guiones (sin wildcards ni regex).
func isSimpleMatcher(matcher string) bool {
	for _, r := range matcher {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '|' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// ParseGlobMatcher parsea un matcher tipo "ToolName(pattern)" en sus partes.
// Retorna el nombre de herramienta y el patrón glob.
// Si no tiene paréntesis, retorna el matcher completo y "".
func ParseGlobMatcher(matcher string) (toolName, globPattern string) {
	idx := strings.Index(matcher, "(")
	if idx <= 0 {
		return matcher, ""
	}
	toolName = matcher[:idx]
	globPattern = strings.TrimSuffix(matcher[idx+1:], ")")
	return toolName, globPattern
}
