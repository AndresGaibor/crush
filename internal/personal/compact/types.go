package compact

import "time"

// CompactAction indica qué acción tomó el sistema.
type CompactAction string

const (
	ActionNone               CompactAction = "none"
	ActionMicroCompact       CompactAction = "micro_compact"
	ActionMemoryCompact      CompactAction = "memory_compact"
	ActionAutoCompact        CompactAction = "auto_compact"
	ActionAutoCompactSkipped CompactAction = "auto_compact_skipped"
)

// CompactLevel indica los niveles disponibles.
type CompactLevel int

const (
	LevelOff CompactLevel = iota
	LevelMicroCompact
	LevelMemoryCompact
	LevelFull
)

// CompactConfig configura el sistema de compactación.
type CompactConfig struct {
	Level           CompactLevel `json:"level"`
	ThresholdPct    float64      `json:"threshold_pct"`
	MinTokensKeep   int          `json:"min_tokens_keep,omitempty"`
	BufferTokens    int          `json:"buffer_tokens,omitempty"`
	MaxOutputTokens int          `json:"max_output_tokens,omitempty"`
	MicroMaxLines   int          `json:"micro_max_lines,omitempty"`
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() *CompactConfig {
	return &CompactConfig{
		Level:           LevelMicroCompact,
		ThresholdPct:    0.80,
		MinTokensKeep:   10_000,
		BufferTokens:    13_000,
		MaxOutputTokens: 20_000,
		MicroMaxLines:   200,
	}
}

// CompactResult contiene el resultado de una operación de compactación.
type CompactResult struct {
	Action       CompactAction `json:"action"`
	TokensBefore int           `json:"tokens_before"`
	TokensAfter  int           `json:"tokens_after"`
	TokensSaved  int           `json:"tokens_saved"`
	Duration     time.Duration `json:"duration"`
	Details      string        `json:"details,omitempty"`
}

// MicroCompactRule transforma contenido textual.
type MicroCompactRule func(content string) string

// CompactableTools agrupa herramientas con resultados que conviene truncar.
var CompactableTools = map[string]bool{
	"Bash":        true,
	"View":        true,
	"Read":        true,
	"Grep":        true,
	"Glob":        true,
	"Fetch":       true,
	"Ls":          true,
	"Download":    true,
	"Sourcegraph": true,
}

// CompactableToolsPreserveInput preserva input además del output.
var CompactableToolsPreserveInput = map[string]bool{
	"Edit":      true,
	"MultiEdit": true,
	"Write":     true,
}
