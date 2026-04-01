package memory

import (
	"fmt"
	"strings"
	"time"
)

// ToolInput es el input de la herramienta memory.
type ToolInput struct {
	Action  string   `json:"action"`            // "save", "load", "delete", "list", "search", "stats", "suggest"
	ID      string   `json:"id,omitempty"`      // ID de la memoria (para save/load/delete)
	Content string   `json:"content,omitempty"` // Contenido (para save)
	Scope   string   `json:"scope,omitempty"`   // "project" o "global" (para save, default "project")
	Tags    []string `json:"tags,omitempty"`    // Tags (para save)
	Query   string   `json:"query,omitempty"`   // Query de búsqueda (para search)
}

// ToolOutput es el resultado de la herramienta memory.
type ToolOutput struct {
	Result string `json:"result"`
}

// Execute ejecuta una acción de la herramienta memory.
func Execute(mgr *MemoryManager, detector *PatternDetector, input ToolInput) (ToolOutput, error) {
	switch strings.ToLower(input.Action) {
	case "save":
		if input.ID == "" {
			return ToolOutput{Result: "Error: id is required for save action"}, nil
		}
		if input.Content == "" {
			return ToolOutput{Result: "Error: content is required for save action"}, nil
		}
		scope := ScopeProject
		if strings.ToLower(input.Scope) == "global" {
			scope = ScopeGlobal
		}
		mem, err := mgr.Save(input.ID, input.Content, scope, input.Tags)
		if err != nil {
			return ToolOutput{Result: fmt.Sprintf("Error saving memory: %v", err)}, nil
		}
		return ToolOutput{Result: fmt.Sprintf("Memory saved: %s (%s, %d bytes)", mem.ID, mem.Scope, mem.Size)}, nil

	case "load":
		if input.ID == "" {
			return ToolOutput{Result: "Error: id is required for load action"}, nil
		}
		mem, err := mgr.Load(input.ID)
		if err != nil {
			return ToolOutput{Result: fmt.Sprintf("Error: %v", err)}, nil
		}
		return ToolOutput{Result: fmt.Sprintf("Memory: %s\n\n%s", mem.ID, mem.Content)}, nil

	case "delete":
		if input.ID == "" {
			return ToolOutput{Result: "Error: id is required for delete action"}, nil
		}
		if err := mgr.Delete(input.ID); err != nil {
			return ToolOutput{Result: fmt.Sprintf("Error: %v", err)}, nil
		}
		return ToolOutput{Result: fmt.Sprintf("Memory deleted: %s", input.ID)}, nil

	case "list":
		memories, err := mgr.All()
		if err != nil {
			return ToolOutput{Result: fmt.Sprintf("Error: %v", err)}, nil
		}
		if len(memories) == 0 {
			return ToolOutput{Result: "No memories found"}, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d memories:\n\n", len(memories)))
		for _, mem := range memories {
			icon := "[P]"
			if mem.Scope == ScopeGlobal {
				icon = "[G]"
			}
			sb.WriteString(fmt.Sprintf("  %s %s (%s, %s, %d bytes)\n",
				icon, mem.ID, mem.Scope, mem.UpdatedAt.Format("2006-01-02"), mem.Size))
			if len(mem.Tags) > 0 {
				sb.WriteString(fmt.Sprintf("      tags: %s\n", strings.Join(mem.Tags, ", ")))
			}
		}
		return ToolOutput{Result: sb.String()}, nil

	case "search":
		if input.Query == "" {
			return ToolOutput{Result: "Error: query is required for search action"}, nil
		}
		scanner := NewScanner(mgr)
		results, err := scanner.FindRelevant(input.Query, 5)
		if err != nil {
			return ToolOutput{Result: fmt.Sprintf("Error: %v", err)}, nil
		}
		if len(results) == 0 {
			return ToolOutput{Result: "No relevant memories found"}, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d relevant memories for '%s':\n\n", len(results), input.Query))
		for _, mem := range results {
			content, _ := mem.GetContent()
			// Mostrar solo las primeras 3 líneas como preview
			lines := strings.Split(content, "\n")
			preview := lines
			if len(preview) > 4 {
				preview = preview[:4]
				preview = append(preview, "...")
			}
			sb.WriteString(fmt.Sprintf("  [%s] %s (%s)\n", mem.Scope, mem.ID, mem.UpdatedAt.Format("2006-01-02")))
			for _, line := range preview {
				sb.WriteString(fmt.Sprintf("    %s\n", line))
			}
			sb.WriteString("\n")
		}
		return ToolOutput{Result: sb.String()}, nil

	case "stats":
		ager := NewAger(mgr, 90*24*time.Hour, false) // 90 días por defecto
		stats := ager.Stats()
		return ToolOutput{Result: fmt.Sprintf(
			"Memory stats:\n  Total: %d\n  Project: %d\n  Global: %d\n  Stale (>90d): %d\n  Total size: %d bytes",
			stats.Total, stats.Project, stats.Global, stats.Stale, stats.TotalSize,
		)}, nil

	case "suggest":
		if detector == nil {
			return ToolOutput{Result: "Pattern detector not initialized"}, nil
		}
		suggestions := detector.CheckSuggestions()
		if len(suggestions) == 0 {
			return ToolOutput{Result: "No patterns detected yet. Keep using Crush and patterns will be identified automatically."}, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d suggested memories:\n\n", len(suggestions)))
		for _, s := range suggestions {
			sb.WriteString(fmt.Sprintf("  [%s] (%dx) %s\n", s.Category, s.Count, s.Pattern))
		}
		return ToolOutput{Result: sb.String()}, nil

	default:
		return ToolOutput{Result: fmt.Sprintf(
			"Unknown action: %s. Valid actions: save, load, delete, list, search, stats, suggest",
			input.Action,
		)}, nil
	}
}
