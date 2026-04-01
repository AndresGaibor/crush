package tools

import (
	"context"
	"testing"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/config"
)

type fakeConfigEditor struct {
	cfg *config.Config
}

func (f *fakeConfigEditor) Config() *config.Config { return f.cfg }
func (f *fakeConfigEditor) SetConfigField(scope config.Scope, key string, value any) error {
	return nil
}
func (f *fakeConfigEditor) SetCompactMode(scope config.Scope, enabled bool) error {
	f.cfg.Options.TUI.CompactMode = enabled
	return nil
}
func (f *fakeConfigEditor) SetTransparentBackground(scope config.Scope, enabled bool) error {
	f.cfg.Options.TUI.Transparent = &enabled
	return nil
}

func TestBuildConfigTool(t *testing.T) {
	enabled := false
	editor := &fakeConfigEditor{
		cfg: &config.Config{
			Options: &config.Options{
				TUI: &config.TUIOptions{
					CompactMode: true,
					Transparent: &enabled,
				},
				DisableNotifications: false,
			},
		},
	}

	tool := BuildConfigTool(editor)
	resp, err := tool.Run(context.Background(), fantasy.ToolCall{Input: `{"action":"get","key":"options.tui.compact_mode"}`})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if resp.IsError {
		t.Fatalf("expected success, got error: %s", resp.Content)
	}

	resp, err = tool.Run(context.Background(), fantasy.ToolCall{Input: `{"action":"set","key":"options.tui.compact_mode","value":false}`})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if resp.IsError {
		t.Fatalf("expected success, got error: %s", resp.Content)
	}
	if editor.cfg.Options.TUI.CompactMode {
		t.Fatal("expected compact mode to be updated")
	}
}
