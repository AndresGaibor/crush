package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"charm.land/fantasy"
	appconfig "github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/personal/hooks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyToConfig_MergesHooksSkillsAndMCP(t *testing.T) {
	Reset()
	hooks.Reset()

	tmpDir := t.TempDir()
	t.Setenv("CRUSH_GLOBAL_CONFIG", filepath.Join(tmpDir, "global-config"))
	t.Setenv("CRUSH_GLOBAL_DATA", filepath.Join(tmpDir, "global-data"))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "global-config"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "global-data"), 0o755))

	pluginRoot := filepath.Join(tmpDir, "demo-plugin")
	require.NoError(t, os.MkdirAll(filepath.Join(pluginRoot, "tools"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pluginRoot, "skills", "deploy"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pluginRoot, "hooks"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(pluginRoot, "plugin.json"), []byte(`{
		"name": "demo-plugin",
		"version": "1.0.0",
		"description": "Demo plugin",
		"tools": [{
			"name": "InspectFilesystem",
			"description": "Inspect the filesystem",
			"source": "./tools/inspect.md",
			"inputSchema": {
				"type": "object",
				"properties": {
					"path": {
						"type": "string",
						"description": "Path to inspect"
					}
				},
				"required": ["path"]
			}
		}],
		"skills": ["./skills/deploy"],
		"hooks": {
			"PostToolUse": [{
				"matcher": "*",
				"command": "echo plugin-hook"
			}]
		},
		"mcpServers": {
			"filesystem": {
				"type": "stdio",
				"command": "node",
				"args": ["server.js"]
			}
		}
	}`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(pluginRoot, "tools", "inspect.md"), []byte(`---
description: Inspect the filesystem
---

# InspectFilesystem

Inspect the filesystem.
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(pluginRoot, "skills", "deploy", "SKILL.md"), []byte(`---
name: deploy
description: Deployment skill
---

Use this skill to deploy safely.
`), 0o644))

	hookConfig := hooks.HookConfigMap{
		hooks.PostToolUse: {
			{Matcher: "Write|Edit", Command: "echo user-hook"},
		},
	}
	hookMgr := hooks.Init(tmpDir, hookConfig)

	manager, err := Init(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, manager)

	cfg := &appconfig.Config{
		MCP: appconfig.MCPs{
			"user-mcp": {Type: appconfig.MCPStdio, Command: "echo"},
		},
		Options: &appconfig.Options{
			SkillsPaths: []string{filepath.Join(tmpDir, "existing-skills")},
		},
	}
	cfg.SetupAgents()

	manager.ApplyToConfig(cfg, hookMgr)

	assert.Contains(t, cfg.Options.SkillsPaths, filepath.Join(pluginRoot, "skills", "deploy"))
	assert.Contains(t, cfg.Agents[appconfig.AgentCoder].AllowedTools, "InspectFilesystem")
	assert.Contains(t, cfg.Agents[appconfig.AgentTask].AllowedTools, "InspectFilesystem")

	pluginServer, ok := cfg.MCP["plugin:demo-plugin@project:filesystem"]
	require.True(t, ok)
	assert.Equal(t, appconfig.MCPStdio, pluginServer.Type)
	assert.Equal(t, "node", pluginServer.Command)
	assert.Equal(t, []string{"server.js"}, pluginServer.Args)

	hooksCfg := hookMgr.Config()
	require.Len(t, hooksCfg[hooks.PostToolUse], 2)
	assert.Equal(t, "echo user-hook", hooksCfg[hooks.PostToolUse][0].Command)
	assert.Equal(t, "echo plugin-hook", hooksCfg[hooks.PostToolUse][1].Command)
}

func TestManager_ReloadsPlugins(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "demo-plugin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "demo-plugin", "plugin.json"), []byte(`{
		"name": "demo-plugin",
		"description": "Demo plugin",
		"tools": [{
			"name": "FirstTool",
			"source": "./tool.md"
		}]
	}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "demo-plugin", "tool.md"), []byte(`# FirstTool`), 0o644))

	manager, err := Init(tmpDir)
	require.NoError(t, err)
	require.Len(t, manager.Registry.GetTools(), 1)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "second-plugin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "second-plugin", "plugin.json"), []byte(`{
		"name": "second-plugin",
		"description": "Second plugin",
		"tools": [{
			"name": "SecondTool",
			"source": "./tool.md"
		}]
	}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "second-plugin", "tool.md"), []byte(`# SecondTool`), 0o644))

	require.NoError(t, manager.Reload())
	assert.Len(t, manager.Registry.GetTools(), 2)
}

func TestBuildPluginTools_UsesPluginRootAndSchema(t *testing.T) {
	Reset()

	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "demo-plugin", "tools"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "demo-plugin", "plugin.json"), []byte(`{
		"name": "demo-plugin",
		"description": "Demo plugin",
		"tools": [{
			"name": "InspectFilesystem",
			"description": "Inspect the filesystem",
			"source": "./tools/inspect.md",
			"inputSchema": {
				"type": "object",
				"properties": {
					"path": {
						"type": "string",
						"description": "Path to inspect"
					}
				},
				"required": ["path"]
			}
		}]
	}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "demo-plugin", "tools", "inspect.md"), []byte(`---
description: Inspect the filesystem
---

# InspectFilesystem

Inspect the filesystem.
`), 0o644))

	manager, err := Init(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, manager)

	tools, err := BuildPluginTools(manager.Registry.GetTools())
	require.NoError(t, err)
	require.Len(t, tools, 1)

	info := tools[0].Info()
	assert.Equal(t, "InspectFilesystem", info.Name)
	assert.Contains(t, info.Description, "Inspect the filesystem")
	assert.Equal(t, []string{"path"}, info.Required)
	assert.Equal(t, "object", info.Parameters["type"])

	response, err := tools[0].Run(context.Background(), fantasy.ToolCall{
		ID:    "1",
		Name:  "InspectFilesystem",
		Input: `{"path":"./tools"}`,
	})
	require.NoError(t, err)
	assert.Contains(t, response.Content, "demo-plugin")
	assert.Contains(t, response.Content, "tools/inspect.md")
}
