package compact

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/message"
	personalmemory "github.com/charmbracelet/crush/internal/personal/memory"
)

// MemoryCompact extrae memorias importantes antes del resumen.
type MemoryCompact struct {
	memoryMgr *personalmemory.MemoryManager
}

// NewMemoryCompact crea un extractor de memorias.
func NewMemoryCompact(mgr *personalmemory.MemoryManager) *MemoryCompact {
	return &MemoryCompact{memoryMgr: mgr}
}

// ExtractMemories analiza la conversación y guarda memorias sugeridas.
func (mc *MemoryCompact) ExtractMemories(ctx context.Context, msgs []message.Message, sessionID string) (int, error) {
	if mc.memoryMgr == nil {
		return 0, fmt.Errorf("memory manager not initialized")
	}

	detector := personalmemory.GetDetector()
	if detector == nil {
		detector = personalmemory.NewPatternDetector(3)
	}

	observed := 0
	for _, msg := range msgs {
		if msg.Role != message.User {
			continue
		}
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case message.TextContent:
				observed += observeText(detector, p.Text)
			case *message.TextContent:
				if p != nil {
					observed += observeText(detector, p.Text)
				}
			}
		}
	}

	suggestions := detector.CheckSuggestions()
	suggestions = dedupeSuggestions(suggestions)
	extracted := 0
	for _, suggestion := range suggestions {
		content := personalmemory.GenerateMemoryContent(suggestion)
		id := buildMemoryID(sessionID, suggestion.Pattern)
		exists, err := memoryAlreadyCaptured(mc.memoryMgr, id, content, suggestion.Category)
		if err != nil {
			slog.Warn("Failed to inspect existing memories", "id", id, "error", err)
		}
		if exists {
			continue
		}
		if _, err := mc.memoryMgr.Save(id, content, personalmemory.ScopeProject, []string{suggestion.Category, "auto-generated"}); err != nil {
			slog.Warn("Failed to save memory", "id", id, "error", err)
			continue
		}
		extracted++
	}

	if extracted > 0 {
		slog.Info("Memory compact extracted memories", "session_id", sessionID, "count", extracted)
	}

	_ = observed
	return extracted, nil
}

// GetMemoryContext retorna contexto de memorias relevantes para el prompt.
func (mc *MemoryCompact) GetMemoryContext(query string) string {
	if mc.memoryMgr == nil {
		return ""
	}

	scanner := personalmemory.NewScanner(mc.memoryMgr)
	relevant, err := scanner.FindRelevant(query, 10)
	if err != nil || len(relevant) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Persistent Memories\n\n")
	for _, mem := range relevant {
		content, err := mem.GetContent()
		if err != nil {
			continue
		}
		lines := strings.Split(content, "\n")
		preview := lines
		if len(preview) > 5 {
			preview = append(preview[:5], "...")
		}
		sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", mem.ID, mem.Scope, strings.Join(preview, " ")))
	}
	return sb.String()
}

func observeText(detector *personalmemory.PatternDetector, text string) int {
	if detector == nil || text == "" {
		return 0
	}

	observed := 0
	for _, correction := range personalmemory.ExtractCorrectionsFromDiff(text) {
		detector.Observe("preference", correction)
		observed++
	}
	for _, suggestion := range guessPreferences(text) {
		detector.Observe("preference", suggestion)
		observed++
	}
	return observed
}

func guessPreferences(text string) []string {
	var suggestions []string
	lower := strings.ToLower(text)
	for _, marker := range []string{"prefiero", "quiero", "siempre", "usa ", "no uses", "evita ", "cambia ", "reemplaza "} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			value := strings.TrimSpace(text[idx+len(marker):])
			if value != "" {
				suggestions = append(suggestions, value)
			}
		}
	}
	return suggestions
}

func buildMemoryID(sessionID, pattern string) string {
	base := strings.ToLower(sessionID + "-" + pattern)
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.ReplaceAll(base, ":", "-")
	base = strings.ReplaceAll(base, "/", "-")
	if len(base) > 64 {
		base = base[:64]
	}
	base = strings.Trim(base, "-")
	if base == "" {
		return "auto-memory"
	}
	return "auto-" + base
}

func dedupeSuggestions(suggestions []personalmemory.SuggestedMemory) []personalmemory.SuggestedMemory {
	if len(suggestions) <= 1 {
		return suggestions
	}

	seen := make(map[string]struct{}, len(suggestions))
	uniq := make([]personalmemory.SuggestedMemory, 0, len(suggestions))
	for _, suggestion := range suggestions {
		key := normalizeSuggestionKey(suggestion)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		uniq = append(uniq, suggestion)
	}

	sort.SliceStable(uniq, func(i, j int) bool {
		if uniq[i].Count == uniq[j].Count {
			return uniq[i].Pattern < uniq[j].Pattern
		}
		return uniq[i].Count > uniq[j].Count
	})
	return uniq
}

func normalizeSuggestionKey(s personalmemory.SuggestedMemory) string {
	return strings.ToLower(strings.TrimSpace(s.Category) + "|" + strings.TrimSpace(s.Pattern))
}

func memoryAlreadyCaptured(mgr *personalmemory.MemoryManager, id, content, category string) (bool, error) {
	memories, err := mgr.Project()
	if err != nil {
		return false, err
	}

	normalizedContent := normalizeMemoryContent(content)
	normalizedCategory := strings.ToLower(strings.TrimSpace(category))
	for _, mem := range memories {
		if mem.ID == id {
			existing, err := mem.GetContent()
			if err != nil {
				continue
			}
			if normalizeMemoryContent(existing) == normalizedContent {
				return true, nil
			}
		}
		if !hasTag(mem.Tags, normalizedCategory) {
			continue
		}
		existing, err := mem.GetContent()
		if err != nil {
			continue
		}
		if normalizeMemoryContent(existing) == normalizedContent {
			return true, nil
		}
	}

	return false, nil
}

func normalizeMemoryContent(content string) string {
	content = strings.ToLower(content)
	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.Join(strings.Fields(content), " ")
	return content
}

func hasTag(tags []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, tag := range tags {
		if strings.ToLower(strings.TrimSpace(tag)) == target {
			return true
		}
	}
	return false
}

// SessionMessagesToFantasy convierte mensajes locales a un prompt fantasy.
// Se usa únicamente en tests y utilidades de compactación.
func SessionMessagesToFantasy(msgs []message.Message) []fantasy.Message {
	result := make([]fantasy.Message, 0, len(msgs))
	for _, msg := range msgs {
		ai := msg.ToAIMessage()
		result = append(result, ai...)
	}
	return result
}
