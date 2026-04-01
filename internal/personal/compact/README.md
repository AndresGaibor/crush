# Compact System — Intelligent Context Compression

The Compact System implements a multi-layer pipeline for intelligent context compression when conversations grow too large.

## Architecture

```
Level 1: Micro-Compact (no LLM)
  ├─ Truncate repetitive lines
  ├─ Collapse stack traces
  ├─ Limit output lines
  ├─ Strip ANSI escapes
  ├─ Truncate long lines
  └─ Collapse empty lines

Level 2: Memory Compact (small LLM)
  └─ Extract patterns as persistent memories

Level 3: Auto-Compact (full LLM)
  └─ Delegated to existing Summarize() method
```

## Usage

### Configuration

Add to `.crush/crush.json`:

```json
{
  "options": {
    "compact": {
      "level": 1,
      "threshold_pct": 0.80,
      "micro_max_lines": 200,
      "buffer_tokens": 13000,
      "max_output_tokens": 20000,
      "min_tokens_keep": 10000
    }
  }
}
```

### Levels

| Level | Value | Effect |
|-------|-------|--------|
| 0 | `off` | No compact (original Crush behavior) |
| 1 | `micro_compact` | Only micro-compact, no LLM. Recommended to start |
| 2 | `memory_compact` | Micro + extract memories before summary |
| 3 | `full` | Micro + Memories + Auto-compact complete |

## Integration

The Compact System integrates with:

1. **PrepareStep callback** in `agent.go` — applies micro-compact before each LLM call
2. **Summarize() method** in `agent.go` — enriches summary with extracted memories
3. **Session messages** — maintains backward compatibility with existing message history

## Token Estimation

Uses character-based heuristic:
- ~3.5 chars per token for code+text
- ~2000 tokens per image
- 1.2x padding for conservative estimates

## Files

- `types.go` — Core types and constants
- `estimator.go` — Token estimation logic
- `micro.go` — 6 micro-compact rules
- `memory.go` — Memory extraction
- `manager.go` — Pipeline orchestration
- `boundary.go` — Compact boundary markers
- `prompt.go` — Prompt templates
- `config.go` — Configuration loading
- `init.go` — Module initialization
- `*_test.go` — Unit tests

## Testing

```bash
go test -v ./internal/personal/compact/...
go test -race ./internal/personal/compact/...
go test -cover ./internal/personal/compact/...
```
