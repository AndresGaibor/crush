package compact

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/message"
	personalmemory "github.com/charmbracelet/crush/internal/personal/memory"
)

// Manager orquesta la compactación de contexto.
type Manager struct {
	config *CompactConfig
	rules  []MicroCompactRule
}

// NewManager crea un manager con configuración por defecto si hace falta.
func NewManager(config *CompactConfig) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	return &Manager{
		config: config,
		rules:  DefaultRules(config.MicroMaxLines),
	}
}

// Process aplica micro-compactación y evalúa si el contexto se acerca al límite.
func (m *Manager) Process(msgs []fantasy.Message, contextWindow int) *CompactResult {
	start := time.Now()
	tokensBefore := EstimateTokens(msgs)
	if contextWindow <= 0 {
		return &CompactResult{
			Action:       ActionNone,
			TokensBefore: tokensBefore,
			TokensAfter:  tokensBefore,
			Duration:     time.Since(start),
		}
	}

	threshold := m.calcThreshold(contextWindow)
	microSaved := int(0)
	if m.config.Level >= LevelMicroCompact {
		microSaved = ApplyMicroCompact(msgs, m.rules, m.config.MicroMaxLines)
	}
	tokensAfter := EstimateTokens(msgs)
	if tokensAfter < threshold {
		return &CompactResult{
			Action:       ActionMicroCompact,
			TokensBefore: tokensBefore,
			TokensAfter:  tokensAfter,
			TokensSaved:  tokensBefore - tokensAfter,
			Duration:     time.Since(start),
			Details:      fmt.Sprintf("micro compact saved %s tokens", FormatTokens(tokensBefore-tokensAfter)),
		}
	}

	result := &CompactResult{
		Action:       ActionAutoCompact,
		TokensBefore: tokensBefore,
		TokensAfter:  tokensAfter,
		TokensSaved:  microSaved,
		Duration:     time.Since(start),
		Details:      fmt.Sprintf("threshold reached (%s/%s), auto-compact needed", FormatTokens(tokensAfter), FormatTokens(threshold)),
	}
	if m.config.Level == LevelOff {
		result.Action = ActionNone
	}
	return result
}

// ApplyMicroCompactOnly aplica solo la compactación heurística.
func (m *Manager) ApplyMicroCompactOnly(msgs []fantasy.Message) int {
	if m.config.Level < LevelMicroCompact {
		return 0
	}
	return ApplyMicroCompact(msgs, m.rules, m.config.MicroMaxLines)
}

// ShouldAutoCompact devuelve true cuando el contexto supera el umbral.
func (m *Manager) ShouldAutoCompact(msgs []fantasy.Message, contextWindow int) bool {
	return EstimateTokens(msgs) >= m.calcThreshold(contextWindow)
}

// ExtractMemories extrae memorias persistentes si el sistema está listo.
func (m *Manager) ExtractMemories(ctx context.Context, msgs []message.Message, sessionID string) (int, error) {
	if m.config.Level < LevelMemoryCompact {
		return 0, nil
	}

	memoryMgr := personalmemory.GetManager()
	if memoryMgr == nil {
		return 0, nil
	}

	return NewMemoryCompact(memoryMgr).ExtractMemories(ctx, msgs, sessionID)
}

// GetMemorySummaryContext devuelve contexto relevante de memorias para el resumen.
func (m *Manager) GetMemorySummaryContext(query string) string {
	if m.config.Level < LevelMemoryCompact {
		return ""
	}
	memoryMgr := personalmemory.GetManager()
	if memoryMgr == nil {
		return ""
	}
	return NewMemoryCompact(memoryMgr).GetMemoryContext(query)
}

func (m *Manager) calcThreshold(cw int) int {
	effective := cw - m.config.MaxOutputTokens
	if effective < 0 {
		effective = 0
	}
	if cw > 200_000 && m.config.BufferTokens > 0 {
		reserved := cw - m.config.BufferTokens
		if reserved > 0 && reserved < effective {
			effective = reserved
		}
	}
	threshold := int(float64(effective) * m.config.ThresholdPct)
	if threshold < m.config.MinTokensKeep {
		threshold = m.config.MinTokensKeep
	}
	return threshold
}

// Config retorna la configuración activa.
func (m *Manager) Config() *CompactConfig {
	return m.config
}

// LogResult escribe una línea de log consistente para compactación.
func (m *Manager) LogResult(result *CompactResult) {
	if result == nil || result.Action == ActionNone {
		return
	}
	slog.Debug("Compact pipeline evaluated",
		"action", result.Action,
		"saved", FormatTokens(result.TokensSaved),
		"before", FormatTokens(result.TokensBefore),
		"after", FormatTokens(result.TokensAfter),
		"details", result.Details,
	)
}
