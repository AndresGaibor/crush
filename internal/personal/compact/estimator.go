package compact

import (
	"fmt"
	"math"
	"unicode/utf8"

	"charm.land/fantasy"
)

const (
	CharsPerToken  = 3.5
	ImageTokenCost = 2_000
	PaddingFactor  = 1.2
)

// EstimateTokens estima el costo en tokens de un prompt fantasy.
func EstimateTokens(msgs []fantasy.Message) int {
	total := 0
	toolNames := toolNameIndex(msgs)

	for _, msg := range msgs {
		total += estimateMessageTokens(msg, toolNames)
	}

	return int(math.Ceil(float64(total) * PaddingFactor))
}

func estimateMessageTokens(msg fantasy.Message, toolNames map[string]string) int {
	total := 0
	for _, part := range msg.Content {
		total += estimatePartTokens(part, toolNames)
	}
	return total
}

func estimatePartTokens(part fantasy.MessagePart, toolNames map[string]string) int {
	switch p := part.(type) {
	case fantasy.TextPart:
		return textTokens(p.Text)
	case *fantasy.TextPart:
		return textTokens(p.Text)
	case fantasy.ReasoningPart:
		return textTokens(p.Text)
	case *fantasy.ReasoningPart:
		return textTokens(p.Text)
	case fantasy.FilePart:
		return fileTokens(p)
	case *fantasy.FilePart:
		return fileTokens(*p)
	case fantasy.ToolCallPart:
		return textTokens(p.ToolName) + textTokens(p.Input)
	case *fantasy.ToolCallPart:
		return textTokens(p.ToolName) + textTokens(p.Input)
	case fantasy.ToolResultPart:
		return toolResultTokens(p, toolNames)
	case *fantasy.ToolResultPart:
		return toolResultTokens(*p, toolNames)
	default:
		return 0
	}
}

func fileTokens(file fantasy.FilePart) int {
	if file.MediaType != "" && len(file.MediaType) >= 5 && file.MediaType[:5] == "image" {
		return ImageTokenCost
	}
	return int(math.Ceil(float64(utf8.RuneCountInString(string(file.Data))) / CharsPerToken))
}

func toolResultTokens(result fantasy.ToolResultPart, toolNames map[string]string) int {
	total := 0
	if name := toolNames[result.ToolCallID]; name != "" {
		total += textTokens(name)
	}

	switch out := result.Output.(type) {
	case fantasy.ToolResultOutputContentText:
		total += textTokens(out.Text)
	case *fantasy.ToolResultOutputContentText:
		total += textTokens(out.Text)
	case fantasy.ToolResultOutputContentError:
		if out.Error != nil {
			total += textTokens(out.Error.Error())
		}
	case *fantasy.ToolResultOutputContentError:
		if out != nil && out.Error != nil {
			total += textTokens(out.Error.Error())
		}
	case fantasy.ToolResultOutputContentMedia:
		total += textTokens(out.Text)
	case *fantasy.ToolResultOutputContentMedia:
		if out != nil {
			total += textTokens(out.Text)
		}
	}

	return total
}

func toolNameIndex(msgs []fantasy.Message) map[string]string {
	index := make(map[string]string)
	for _, msg := range msgs {
		for _, part := range msg.Content {
			switch p := part.(type) {
			case fantasy.ToolCallPart:
				index[p.ToolCallID] = p.ToolName
			case *fantasy.ToolCallPart:
				index[p.ToolCallID] = p.ToolName
			}
		}
	}
	return index
}

func textTokens(s string) int {
	if s == "" {
		return 0
	}
	return int(math.Ceil(float64(utf8.RuneCountInString(s)) / CharsPerToken))
}

// FormatTokens formatea cantidades grandes de tokens.
func FormatTokens(n int) string {
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
