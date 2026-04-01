package tools

import (
	"context"
	"strings"
	"testing"

	"charm.land/fantasy"
)

func TestBuildToolSearchTool(t *testing.T) {
	grepTool := fantasy.NewAgentTool(
		"Grep",
		"Searches text in files",
		func(ctx context.Context, input struct {
			Query string `json:"query"`
		}, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			return fantasy.NewTextResponse("ok"), nil
		},
	)
	readTool := fantasy.NewAgentTool(
		"Read",
		"Reads files",
		func(ctx context.Context, input struct {
			Path string `json:"path"`
		}, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			return fantasy.NewTextResponse("ok"), nil
		},
	)

	tool := BuildToolSearchTool([]fantasy.AgentTool{grepTool, readTool})
	resp, err := tool.Run(context.Background(), fantasy.ToolCall{
		Input: `{"query":"grep","limit":5}`,
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if resp.IsError {
		t.Fatalf("expected success response, got error: %s", resp.Content)
	}
	if !strings.Contains(resp.Content, "Grep") {
		t.Fatalf("expected Grep in response, got: %s", resp.Content)
	}
}
