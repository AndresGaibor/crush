package compact

import (
	"fmt"
	"time"

	"github.com/charmbracelet/crush/internal/message"
)

// CompactBoundary marca un evento de compactación.
type CompactBoundary struct {
	Trigger      string    `json:"trigger"`
	TokensBefore int       `json:"tokens_before"`
	TokensAfter  int       `json:"tokens_after"`
	TokensSaved  int       `json:"tokens_saved"`
	Timestamp    time.Time `json:"timestamp"`
	Details      string    `json:"details,omitempty"`
}

// CreateBoundaryMessage crea un mensaje marcador para la historia.
func CreateBoundaryMessage(result *CompactResult) message.Message {
	content := fmt.Sprintf(
		"[Context compacted: %s — saved %s tokens (%s → %s)]",
		result.Action,
		FormatTokens(result.TokensSaved),
		FormatTokens(result.TokensBefore),
		FormatTokens(result.TokensAfter),
	)
	return message.Message{
		Role: message.System,
		Parts: []message.ContentPart{
			message.TextContent{Text: content},
		},
	}
}

// IsBoundaryMessage detecta un mensaje marcador.
func IsBoundaryMessage(msg message.Message) bool {
	if msg.Role != message.System {
		return false
	}
	for _, part := range msg.Parts {
		switch p := part.(type) {
		case message.TextContent:
			return len(p.Text) > 0 && p.Text[0] == '['
		case *message.TextContent:
			return p != nil && len(p.Text) > 0 && p.Text[0] == '['
		}
	}
	return false
}
