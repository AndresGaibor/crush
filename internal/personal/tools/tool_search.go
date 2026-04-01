package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"charm.land/fantasy"
)

type toolSearchInput struct {
	Query string `json:"query" description:"Término de búsqueda para nombres, descripciones y esquemas"`
	Limit int    `json:"limit,omitempty" description:"Cantidad máxima de resultados"`
}

// BuildToolSearchTool construye la herramienta de búsqueda de tools.
func BuildToolSearchTool(tools []fantasy.AgentTool) fantasy.AgentTool {
	snapshot := append([]fantasy.AgentTool(nil), tools...)

	return fantasy.NewAgentTool(
		"tool_search",
		`Search available tools by name, description, or schema and return their full input schemas.`,
		func(ctx context.Context, input toolSearchInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = ctx
			_ = call

			query := strings.TrimSpace(strings.ToLower(input.Query))
			if query == "" {
				return fantasy.NewTextErrorResponse("query is required"), nil
			}

			limit := input.Limit
			if limit <= 0 {
				limit = 10
			}

			matches := make([]toolMatch, 0, len(snapshot))
			for _, tool := range snapshot {
				info := tool.Info()
				haystack := buildToolHaystack(info)
				score := toolMatchScore(query, info, haystack)
				if score <= 0 {
					continue
				}
				matches = append(matches, toolMatch{
					Tool:  tool,
					Info:  info,
					Score: score,
				})
			}

			if len(matches) == 0 {
				return fantasy.NewTextResponse(fmt.Sprintf("No tools found matching %q.", input.Query)), nil
			}

			sort.Slice(matches, func(i, j int) bool {
				if matches[i].Score != matches[j].Score {
					return matches[i].Score > matches[j].Score
				}
				return matches[i].Info.Name < matches[j].Info.Name
			})

			if len(matches) > limit {
				matches = matches[:limit]
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("Found %d tool(s) matching %q:\n\n", len(matches), input.Query))

			for _, match := range matches {
				b.WriteString(fmt.Sprintf("### %s\n", match.Info.Name))
				b.WriteString(fmt.Sprintf("- Description: %s\n", match.Info.Description))
				if len(match.Info.Required) > 0 {
					b.WriteString(fmt.Sprintf("- Required: %s\n", strings.Join(match.Info.Required, ", ")))
				}
				if match.Info.Parallel {
					b.WriteString("- Parallel: yes\n")
				}

				schemaJSON, err := json.MarshalIndent(map[string]any{
					"parameters": match.Info.Parameters,
					"required":   match.Info.Required,
					"parallel":   match.Info.Parallel,
				}, "", "  ")
				if err == nil {
					b.WriteString("- Schema:\n")
					b.WriteString("```json\n")
					b.Write(schemaJSON)
					b.WriteString("\n```\n")
				}
				b.WriteString("\n")
			}

			return fantasy.NewTextResponse(b.String()), nil
		},
	)
}

type toolMatch struct {
	Tool  fantasy.AgentTool
	Info  fantasy.ToolInfo
	Score int
}

func buildToolHaystack(info fantasy.ToolInfo) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(info.Name))
	b.WriteString(" ")
	b.WriteString(strings.ToLower(info.Description))
	b.WriteString(" ")

	if data, err := json.Marshal(info.Parameters); err == nil {
		b.WriteString(strings.ToLower(string(data)))
	}
	if len(info.Required) > 0 {
		b.WriteString(" ")
		b.WriteString(strings.ToLower(strings.Join(info.Required, " ")))
	}
	return b.String()
}

func toolMatchScore(query string, info fantasy.ToolInfo, haystack string) int {
	score := 0
	name := strings.ToLower(info.Name)
	desc := strings.ToLower(info.Description)

	switch {
	case query == name:
		score += 100
	case strings.Contains(name, query):
		score += 80
	}
	if strings.Contains(desc, query) {
		score += 40
	}
	if strings.Contains(haystack, query) {
		score += 20
	}
	return score
}
