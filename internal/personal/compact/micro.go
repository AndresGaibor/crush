package compact

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"charm.land/fantasy"
)

const (
	minVerboseLines  = 40
	minVerboseChars  = 2_500
	minTraceLines    = 8
	minRepeatRunSize = 6
)

// ApplyMicroCompact aplica reglas heurísticas sobre los resultados de herramientas.
// Retorna los tokens ahorrados estimados.
func ApplyMicroCompact(msgs []fantasy.Message, rules []MicroCompactRule, maxLines int) (tokensSaved int) {
	before := estimateCompactableTokens(msgs)
	toolNames := toolNameIndex(msgs)

	for i := range msgs {
		for j, part := range msgs[i].Content {
			switch p := part.(type) {
			case fantasy.ToolResultPart:
				msgs[i].Content[j] = applyRulesToToolResult(p, toolNames, rules)
			case *fantasy.ToolResultPart:
				updated := applyRulesToToolResult(*p, toolNames, rules)
				msgs[i].Content[j] = updated
			}
		}
	}

	after := estimateCompactableTokens(msgs)
	if before > after {
		return before - after
	}
	return 0
}

func estimateCompactableTokens(msgs []fantasy.Message) int {
	total := 0
	toolNames := toolNameIndex(msgs)
	for _, msg := range msgs {
		for _, part := range msg.Content {
			switch p := part.(type) {
			case fantasy.ToolResultPart:
				total += toolResultTokens(p, toolNames)
			case *fantasy.ToolResultPart:
				total += toolResultTokens(*p, toolNames)
			}
		}
	}
	return total
}

func applyRulesToToolResult(result fantasy.ToolResultPart, toolNames map[string]string, rules []MicroCompactRule) fantasy.ToolResultPart {
	name := toolNames[result.ToolCallID]
	if !CompactableTools[name] {
		return result
	}

	original := result.Output
	switch out := result.Output.(type) {
	case fantasy.ToolResultOutputContentText:
		text := out.Text
		if !shouldMicroCompact(text) {
			return result
		}
		for _, rule := range rules {
			text = rule(text)
		}
		result.Output = fantasy.ToolResultOutputContentText{Text: text}
	case *fantasy.ToolResultOutputContentText:
		if out != nil {
			text := out.Text
			if !shouldMicroCompact(text) {
				return result
			}
			for _, rule := range rules {
				text = rule(text)
			}
			result.Output = fantasy.ToolResultOutputContentText{Text: text}
		}
	case fantasy.ToolResultOutputContentError:
		if out.Error != nil {
			text := out.Error.Error()
			if !shouldMicroCompact(text) {
				return result
			}
			for _, rule := range rules {
				text = rule(text)
			}
			result.Output = fantasy.ToolResultOutputContentText{Text: text}
		}
	case *fantasy.ToolResultOutputContentError:
		if out != nil && out.Error != nil {
			text := out.Error.Error()
			if !shouldMicroCompact(text) {
				return result
			}
			for _, rule := range rules {
				text = rule(text)
			}
			result.Output = fantasy.ToolResultOutputContentText{Text: text}
		}
	case fantasy.ToolResultOutputContentMedia:
		if out.Text != "" {
			text := out.Text
			if !shouldMicroCompact(text) {
				return result
			}
			for _, rule := range rules {
				text = rule(text)
			}
			out.Text = text
			result.Output = out
		}
	case *fantasy.ToolResultOutputContentMedia:
		if out != nil && out.Text != "" {
			text := out.Text
			if !shouldMicroCompact(text) {
				return result
			}
			for _, rule := range rules {
				text = rule(text)
			}
			out.Text = text
			result.Output = *out
		}
	}

	if original == nil {
		return result
	}
	return result
}

func shouldMicroCompact(content string) bool {
	if content == "" {
		return false
	}

	lines := strings.Count(content, "\n") + 1
	if lines >= minVerboseLines {
		return true
	}
	if utf8.RuneCountInString(content) >= minVerboseChars {
		return true
	}
	if containsTraceBlock(content) {
		return true
	}
	if repeatedLineRun(content) >= minRepeatRunSize {
		return true
	}
	return false
}

// DefaultRules retorna las reglas heurísticas por defecto.
func DefaultRules(maxLines int) []MicroCompactRule {
	return []MicroCompactRule{
		TruncateRepetitiveLines(3),
		CollapseStackTraces(5, 5),
		LimitLines(maxLines),
		StripANSIEscapes(),
		TruncateLongLines(500),
		StripEmptyLines(),
	}
}

// TruncateRepetitiveLines colapsa líneas consecutivas idénticas.
func TruncateRepetitiveLines(maxRepeat int) MicroCompactRule {
	return func(content string) string {
		if content == "" {
			return content
		}

		lines := strings.Split(content, "\n")
		var out []string
		for i := 0; i < len(lines); {
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) == strings.TrimSpace(lines[i]) && strings.TrimSpace(lines[i]) != "" {
				j++
			}
			runLen := j - i
			if runLen <= maxRepeat {
				out = append(out, lines[i:j]...)
			} else {
				out = append(out, lines[i:i+maxRepeat]...)
				out = append(out, fmt.Sprintf("... [%d identical lines collapsed] ...", runLen-maxRepeat))
			}
			i = j
		}
		return strings.Join(out, "\n")
	}
}

// CollapseStackTraces colapsa bloques de trazas de stack.
func CollapseStackTraces(head, tail int) MicroCompactRule {
	return func(content string) string {
		if content == "" {
			return content
		}

		lines := strings.Split(content, "\n")
		var out []string
		for i := 0; i < len(lines); {
			if !isTraceLine(lines[i]) {
				out = append(out, lines[i])
				i++
				continue
			}

			start := i
			for i < len(lines) && isTraceLine(lines[i]) {
				i++
			}
			block := lines[start:i]
			if len(block) <= head+tail {
				out = append(out, block...)
				continue
			}

			out = append(out, block[:head]...)
			out = append(out, fmt.Sprintf("    ... [%d lines collapsed, showing first %d and last %d] ...",
				len(block)-head-tail, head, tail))
			out = append(out, block[len(block)-tail:]...)
		}

		return strings.Join(out, "\n")
	}
}

// LimitLines trunca a maxLines líneas.
func LimitLines(maxLines int) MicroCompactRule {
	return func(content string) string {
		if content == "" || maxLines <= 0 {
			return content
		}
		lines := strings.Split(content, "\n")
		if len(lines) <= maxLines {
			return content
		}
		truncated := len(lines) - maxLines
		return strings.Join(lines[:maxLines], "\n") +
			fmt.Sprintf("\n\n... [%d lines truncated, showing first %d of %d] ...", truncated, maxLines, len(lines))
	}
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSIEscapes remueve secuencias ANSI.
func StripANSIEscapes() MicroCompactRule {
	return func(content string) string {
		if content == "" {
			return content
		}
		return ansiRE.ReplaceAllString(content, "")
	}
}

// TruncateLongLines trunca líneas individuales largas.
func TruncateLongLines(maxChars int) MicroCompactRule {
	return func(content string) string {
		if content == "" || maxChars <= 0 {
			return content
		}

		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if utf8.RuneCountInString(line) <= maxChars {
				continue
			}
			runes := []rune(line)
			lines[i] = string(runes[:maxChars]) +
				fmt.Sprintf("... [%d chars truncated] ...", utf8.RuneCountInString(line)-maxChars)
		}
		return strings.Join(lines, "\n")
	}
}

// StripEmptyLines limita líneas vacías consecutivas a 2.
func StripEmptyLines() MicroCompactRule {
	return func(content string) string {
		if content == "" {
			return content
		}

		lines := strings.Split(content, "\n")
		var out []string
		emptyCount := 0
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				emptyCount++
				if emptyCount <= 2 {
					out = append(out, line)
				}
				continue
			}

			emptyCount = 0
			out = append(out, line)
		}
		return strings.Join(out, "\n")
	}
}

var tracePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\s+at\s+\S+\.\S+\(.*\)$`),
	regexp.MustCompile(`^\s+created by\s+\S+`),
	regexp.MustCompile(`^\s+goroutine\s+\d+\s+\[`),
	regexp.MustCompile(`^#\d+\s+\d+\s+0x[0-9a-f]+\s+`),
	regexp.MustCompile(`^\t[a-zA-Z_\$][a-zA-Z0-9_\$]*\.`),
	regexp.MustCompile(`^Error:\s+`),
	regexp.MustCompile(`^panic:\s+`),
}

func isTraceLine(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return false
	}
	for _, re := range tracePatterns {
		if re.MatchString(t) {
			return true
		}
	}
	return false
}

func containsTraceBlock(content string) bool {
	lines := strings.Split(content, "\n")
	traceRun := 0
	for _, line := range lines {
		if isTraceLine(line) {
			traceRun++
			if traceRun >= minTraceLines {
				return true
			}
			continue
		}
		traceRun = 0
	}
	return false
}

func repeatedLineRun(content string) int {
	lines := strings.Split(content, "\n")
	maxRun := 1
	run := 1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" && strings.TrimSpace(lines[i]) == strings.TrimSpace(lines[i-1]) {
			run++
			if run > maxRun {
				maxRun = run
			}
			continue
		}
		run = 1
	}
	return maxRun
}
