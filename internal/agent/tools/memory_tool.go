package tools

import (
	"context"
	_ "embed"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/personal/memory"
)

//go:embed memory.md
var memoryDescription []byte

// MemoryParams son los parámetros de la herramienta memory.
type MemoryParams struct {
	Action  string   `json:"action" description:"Action to perform: save, load, delete, list, search, stats, suggest"`
	ID      string   `json:"id,omitempty" description:"Memory ID (for save/load/delete)"`
	Content string   `json:"content,omitempty" description:"Memory content in Markdown (for save)"`
	Scope   string   `json:"scope,omitempty" description:"Scope: project or global (default: project)"`
	Tags    []string `json:"tags,omitempty" description:"Tags for categorization (for save)"`
	Query   string   `json:"query,omitempty" description:"Search query (for search)"`
}

const MemoryToolName = "memory"

// NewMemoryTool creates a new memory tool for persistent memory management.
func NewMemoryTool(workingDir string) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		MemoryToolName,
		string(memoryDescription),
		func(ctx context.Context, params MemoryParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			// Inicializar el sistema de memoria si no está inicializado
			mgr := memory.GetManager()
			if mgr == nil {
				_, err := memory.Init(workingDir)
				if err != nil {
					return fantasy.NewTextErrorResponse("Failed to initialize memory system: " + err.Error()), nil
				}
				mgr = memory.GetManager()
			}

			detector := memory.GetDetector()

			// Ejecutar la acción
			input := memory.ToolInput{
				Action:  params.Action,
				ID:      params.ID,
				Content: params.Content,
				Scope:   params.Scope,
				Tags:    params.Tags,
				Query:   params.Query,
			}

			output, err := memory.Execute(mgr, detector, input)
			if err != nil {
				return fantasy.NewTextErrorResponse("Error: " + err.Error()), nil
			}

			return fantasy.NewTextResponse(output.Result), nil
		},
	)
}
