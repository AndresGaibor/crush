package compact

import (
	"strings"
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/assert"
)

func TestTruncateRepetitiveLines(t *testing.T) {
	t.Parallel()
	rule := TruncateRepetitiveLines(2)
	input := "line1\nline1\nline1\nline1\nline1\nline2"
	result := rule(input)
	assert.Contains(t, result, "line1")
	assert.Contains(t, result, "line2")
	assert.Contains(t, result, "collapsed")

	lines := strings.Split(result, "\n")
	line1Count := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "line1" {
			line1Count++
		}
	}
	assert.Equal(t, 2, line1Count)
}

func TestCollapseStackTraces(t *testing.T) {
	t.Parallel()
	rule := CollapseStackTraces(2, 2)
	input := "Error: something failed\n" +
		"\tat main.main (/main.go:10)\n" +
		"\tat runtime.main (/runtime.go:20)\n" +
		"\tat some.func (/lib.go:30)\n" +
		"\tat another.func (/lib.go:40)\n" +
		"End of trace"
	result := rule(input)
	assert.Contains(t, result, "Error: something failed")
	assert.Contains(t, result, "End of trace")
}

func TestLimitLines(t *testing.T) {
	t.Parallel()
	rule := LimitLines(3)
	input := "line1\nline2\nline3\nline4\nline5"
	result := rule(input)
	assert.Contains(t, result, "line1")
	assert.Contains(t, result, "line3")
	assert.NotContains(t, result, "line4")
	assert.Contains(t, result, "2 lines truncated")
}

func TestStripANSIEscapes(t *testing.T) {
	t.Parallel()
	rule := StripANSIEscapes()
	input := "\x1b[32mgreen text\x1b[0m\nnormal"
	result := rule(input)
	assert.Equal(t, "green text\nnormal", result)
}

func TestTruncateLongLines(t *testing.T) {
	t.Parallel()
	rule := TruncateLongLines(10)
	input := "short\nthis is a very very very very very long line\nanother"
	result := rule(input)
	assert.Contains(t, result, "short")
	assert.Contains(t, result, "truncated")
	assert.Contains(t, result, "another")
}

func TestStripEmptyLines(t *testing.T) {
	t.Parallel()
	rule := StripEmptyLines()
	input := "text1\n\n\n\n\n\ntext2"
	result := rule(input)
	lines := strings.Split(result, "\n")

	prevEmpty := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			prevEmpty++
			continue
		}
		assert.LessOrEqual(t, prevEmpty, 2, "too many empty lines: %v", result)
		prevEmpty = 0
	}
}

func TestApplyMicroCompact(t *testing.T) {
	t.Parallel()
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
					Output: fantasy.ToolResultOutputContentText{
						Text: strings.Repeat("line\n", 40),
					},
				},
			},
		},
	}

	saved := ApplyMicroCompact(msgs, DefaultRules(10), 10)
	assert.Greater(t, saved, 0)
	if tr, ok := msgs[1].Content[0].(fantasy.ToolResultPart); ok {
		text := tr.Output.(fantasy.ToolResultOutputContentText).Text
		assert.True(t, strings.Contains(text, "truncated") || strings.Contains(text, "collapsed"))
	} else {
		t.Fatalf("expected ToolResultPart")
	}
}

func TestApplyMicroCompactSkipsSmallOutput(t *testing.T) {
	t.Parallel()
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
					Output: fantasy.ToolResultOutputContentText{
						Text: "ok\nline2\nline3",
					},
				},
			},
		},
	}

	before := msgs[1].Content[0].(fantasy.ToolResultPart).Output.(fantasy.ToolResultOutputContentText).Text
	saved := ApplyMicroCompact(msgs, DefaultRules(10), 10)
	after := msgs[1].Content[0].(fantasy.ToolResultPart).Output.(fantasy.ToolResultOutputContentText).Text
	assert.Equal(t, 0, saved)
	assert.Equal(t, before, after)
}
