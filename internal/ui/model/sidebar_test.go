package model

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/crush/internal/personal/memory"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/x/ansi"
)

func TestGetDynamicHeightLimitsPrioritizesFiles(t *testing.T) {
	t.Parallel()

	files, memories, lsps, mcps := getDynamicHeightLimits(5)

	if files <= memories {
		t.Fatalf("expected files to receive priority over memory, got files=%d memories=%d", files, memories)
	}
	if files == 0 || memories == 0 || lsps == 0 || mcps == 0 {
		t.Fatalf("expected all sections to stay visible, got files=%d memories=%d lsps=%d mcps=%d", files, memories, lsps, mcps)
	}
}

func TestBuildMemorySectionBodyCompact(t *testing.T) {
	t.Parallel()

	styles := common.DefaultCommon(nil).Styles
	body := buildMemorySectionBody(styles, 24, 1, memory.MemoryStats{Total: 12, Project: 8, Global: 4, Stale: 1}, []memory.Memory{
		{ID: "go-conventions", Scope: memory.ScopeProject, Tags: []string{"go", "idioms"}},
	})

	if len(body) != 1 {
		t.Fatalf("expected compact body to stay at one line, got %d lines: %#v", len(body), body)
	}
	if got := ansi.Strip(body[0]); !strings.Contains(got, "[P]") || !strings.Contains(got, "go-conventions") {
		t.Fatalf("expected compact memory line without tags, got %q", got)
	}
}

func TestBuildMemorySectionBodyMediumWithRecentMemories(t *testing.T) {
	t.Parallel()

	recent := []memory.Memory{
		{ID: "go-conventions", Scope: memory.ScopeProject, Tags: []string{"go", "idioms"}},
		{ID: "release-process", Scope: memory.ScopeGlobal, Tags: []string{"ops", "release", "checklist"}},
		{ID: "ignored-extra", Scope: memory.ScopeProject, Tags: []string{"extra"}},
	}

	styles := common.DefaultCommon(nil).Styles
	body := buildMemorySectionBody(styles, 48, 2, memory.MemoryStats{Total: 2, Project: 1, Global: 1}, recent)

	if len(body) != 4 {
		t.Fatalf("expected two two-line memories, got %d lines: %#v", len(body), body)
	}
	if got := ansi.Strip(body[0]); !strings.Contains(got, "[P]") || !strings.Contains(got, "go-conventions") {
		t.Fatalf("expected first memory to include scope and tags, got %q", got)
	}
	if got := ansi.Strip(body[1]); !strings.Contains(got, "go, idioms") {
		t.Fatalf("expected first memory tags line, got %q", got)
	}
	if got := ansi.Strip(body[2]); !strings.Contains(got, "[G]") || !strings.Contains(got, "release-process") {
		t.Fatalf("expected second memory to include scope and title, got %q", got)
	}
	if got := ansi.Strip(body[3]); !strings.Contains(got, "ops, release") {
		t.Fatalf("expected second memory tags line, got %q", got)
	}
	for _, line := range body {
		if strings.Contains(line, "ignored-extra") {
			t.Fatalf("expected medium density to cap visible memories, got %#v", body)
		}
	}
}

func TestBuildMemorySectionBodyLargeTruncatesLongTags(t *testing.T) {
	t.Parallel()

	recent := []memory.Memory{
		{
			ID:    "very-long-memory-name-that-should-be-truncated",
			Scope: memory.ScopeGlobal,
			Tags: []string{
				"very-long-tag-name-one",
				"very-long-tag-name-two",
				"very-long-tag-name-three",
			},
			UpdatedAt: time.Now(),
		},
		{
			ID:    "project-memory",
			Scope: memory.ScopeProject,
			Tags:  []string{"a", "b"},
		},
		{
			ID:    "extra-memory",
			Scope: memory.ScopeProject,
			Tags:  []string{"c"},
		},
		{
			ID:    "another-memory",
			Scope: memory.ScopeGlobal,
			Tags:  []string{"d"},
		},
		{
			ID:    "overflow-memory",
			Scope: memory.ScopeGlobal,
			Tags:  []string{"e"},
		},
	}

	styles := common.DefaultCommon(nil).Styles
	body := buildMemorySectionBody(styles, 72, 4, memory.MemoryStats{Total: 1, Global: 1}, recent)

	if len(body) != 8 {
		t.Fatalf("expected four two-line memories, got %d lines: %#v", len(body), body)
	}
	if ansi.StringWidth(ansi.Strip(body[0])) > 72 {
		t.Fatalf("expected memory line to fit width, got width %d for %q", ansi.StringWidth(ansi.Strip(body[0])), body[0])
	}
	if !strings.Contains(ansi.Strip(body[0]), "[G]") {
		t.Fatalf("expected global scope prefix, got %q", body[0])
	}
	for _, line := range body {
		if strings.Contains(line, "overflow-memory") {
			t.Fatalf("expected large density to cap visible memories, got %#v", body)
		}
	}
}

func TestBuildMemorySectionBodyEmptyStateUsesSingleMessageWhenNotCompact(t *testing.T) {
	t.Parallel()

	body := buildMemorySectionBody(common.DefaultCommon(nil).Styles, 48, 2, memory.MemoryStats{Total: 0}, nil)

	if len(body) != 1 {
		t.Fatalf("expected empty-state message, got %d lines: %#v", len(body), body)
	}
	if got := ansi.Strip(body[0]); !strings.Contains(got, "No memories yet") {
		t.Fatalf("expected empty-state message, got %q", body[0])
	}
}
