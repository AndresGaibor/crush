package compact

import (
	"context"
	"strings"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/stretchr/testify/assert"
)

func TestProcess_WithValidConfig(t *testing.T) {
	t.Parallel()
	mgr := NewManager(&CompactConfig{
		Level:           LevelMicroCompact,
		ThresholdPct:    0.80,
		MicroMaxLines:   100,
		MaxOutputTokens: 20_000,
	})

	msgs := []fantasy.Message{
		{
			Role: fantasy.MessageRoleUser,
			Content: []fantasy.MessagePart{
				fantasy.TextPart{Text: "x"},
			},
		},
	}

	result := mgr.Process(msgs, 200_000)
	assert.NotNil(t, result)
	assert.True(t, len(string(result.Action)) > 0)
}

func TestProcess_MicroCompactApplied(t *testing.T) {
	t.Parallel()
	mgr := NewManager(&CompactConfig{
		Level:           LevelMicroCompact,
		ThresholdPct:    0.001,
		MicroMaxLines:   2,
		MaxOutputTokens: 20_000,
	})

	largeContent := strings.Repeat("line of text content\n", 100)
	msgs := []fantasy.Message{
		{
			Role: fantasy.MessageRoleAssistant,
			Content: []fantasy.MessagePart{
				fantasy.ToolCallPart{ToolCallID: "1", ToolName: "Bash", Input: "{}"},
			},
		},
		{
			Role: fantasy.MessageRoleTool,
			Content: []fantasy.MessagePart{
				fantasy.ToolResultPart{
					ToolCallID: "1",
					Output:     fantasy.ToolResultOutputContentText{Text: largeContent},
				},
			},
		},
	}

	result := mgr.Process(msgs, 10_000)
	assert.NotEqual(t, ActionNone, result.Action)
}

func TestThresholdCalculation(t *testing.T) {
	t.Parallel()
	mgr := NewManager(&CompactConfig{
		ThresholdPct:    0.80,
		MaxOutputTokens: 20_000,
	})

	assert.Equal(t, 144000, mgr.calcThreshold(200000))
	assert.Equal(t, 64000, mgr.calcThreshold(100000))
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()
	config := DefaultConfig()
	assert.Equal(t, LevelMicroCompact, config.Level)
	assert.Equal(t, 0.80, config.ThresholdPct)
	assert.Equal(t, 200, config.MicroMaxLines)
	assert.Equal(t, 20_000, config.MaxOutputTokens)
}

func TestExtractMemories_NoManager(t *testing.T) {
	t.Parallel()
	mgr := NewManager(DefaultConfig())
	count, err := mgr.ExtractMemories(context.Background(), nil, "test")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestBoundaryMessage(t *testing.T) {
	t.Parallel()
	msg := CreateBoundaryMessage(&CompactResult{
		Action:       ActionMicroCompact,
		TokensBefore: 1000,
		TokensAfter:  500,
		TokensSaved:  500,
	})
	assert.Equal(t, message.System, msg.Role)
	assert.True(t, IsBoundaryMessage(msg))
}
