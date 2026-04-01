package prompt

import "testing"

func TestAppendUniqueContextFilesPreservesOrderAndDeduplicates(t *testing.T) {
	t.Parallel()

	first := []ContextFile{
		{Path: "/tmp/a.md", Content: "a"},
		{Path: "/tmp/b.md", Content: "b"},
	}
	second := []ContextFile{
		{Path: "/tmp/b.md", Content: "dup"},
		{Path: "/tmp/c.md", Content: "c"},
	}

	merged, seen := appendUniqueContextFiles(first, nil, second)
	if len(merged) != 3 {
		t.Fatalf("expected 3 files, got %d", len(merged))
	}
	if merged[0].Path != "/tmp/a.md" || merged[1].Path != "/tmp/b.md" || merged[2].Path != "/tmp/c.md" {
		t.Fatalf("unexpected order: %#v", merged)
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 seen entries, got %d", len(seen))
	}
}
