# Plugin System

The plugin system allows extending Crush with custom tools, hooks, and skills without modifying core code.

## Quick Start

Plugins are defined by a `plugin.json` manifest file in a directory. Crush discovers plugins from:

1. **Project scope**: `.claude-plugin/` directory in the project root
2. **Global scope**: `~/.config/crush/plugins/` directory
3. **Config scope**: Paths specified in `crush.json`

### Plugin Directory Structure

```
.claude-plugin/
├── plugin.json              # Plugin manifest
├── tools/
│   └── my-tool.md          # Tool definitions
├── skills/
│   └── deploy/
│       └── SKILL.md        # Skill file
├── hooks/
│   └── hooks.json          # Hook configuration
└── README.md               # Optional documentation
```

## Plugin Manifest (`plugin.json`)

### Minimal Example

```json
{
  "name": "my-plugin",
  "version": "1.0.0",
  "description": "A simple plugin",
  "author": "Your Name",
  "license": "MIT"
}
```

### Full Example

```json
{
  "name": "code-formatter",
  "version": "1.0.0",
  "description": "Auto-formats code files",
  "author": "Your Name",
  "license": "MIT",
  "keywords": ["formatting", "prettier", "gofmt"],
  
  "tools": [
    {
      "name": "FormatCode",
      "description": "Format code files",
      "source": "./tools/format-code.md",
      "inputSchema": {
        "type": "object",
        "properties": {
          "file_path": {
            "type": "string",
            "description": "Path to format"
          }
        },
        "required": ["file_path"]
      }
    }
  ],
  
  "skills": ["./skills/deploy"],
  
  "hooks": "./hooks/hooks.json"
}
```

## Tool Definition (Markdown)

Tools are defined in Markdown files with YAML frontmatter:

```markdown
---
name: FormatCode
description: Format code files using configured formatter
---

# FormatCode Tool

When asked to format code:

1. Determine formatter from file extension
   - `.go` → `gofmt -w <file>`
   - `.ts`, `.js` → `prettier --write <file>`
   - `.py` → `black <file>`

2. Run the formatter on the file path

3. Report results to the user
```

## Hook Configuration

Plugins can register hooks that fire on tool events:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "gofmt -w $CRUSH_PROJECT_DIR/$(echo $TOOL_INPUT | jq -r '.file_path // empty') 2>/dev/null || true",
            "timeout": 10000
          }
        ]
      }
    ]
  }
}
```

## Configuration

Enable/disable plugins in `crush.json`:

```json
{
  "plugins": {
    "enabled_plugins": {
      "code-formatter@project": true,
      "old-plugin@global": false
    },
    "plugin_options": {
      "code-formatter@project": {
        "formatter": "gofmt",
        "timeout": 15000
      }
    }
  }
}
```

## Plugin Naming

- **Name**: kebab-case, 2-64 characters (required)
- **Version**: Semantic versioning (optional but recommended)
- **ID format**: `{name}@{scope}` (e.g., `my-plugin@project`)

## Discovery & Loading

Plugins are discovered and loaded during application startup:

1. Scan project `.claude-plugin/` directory
2. Scan global `~/.config/crush/plugins/` directory
3. Load manifests and validate
4. Register extensions (tools, hooks, skills)
5. Merge configurations

Invalid plugins are logged but don't crash the system.

## Development

### Create a Plugin

```bash
mkdir -p .claude-plugin/tools .claude-plugin/hooks
cat > .claude-plugin/plugin.json << 'EOF'
{
  "name": "my-plugin",
  "version": "1.0.0",
  "description": "My custom plugin",
  "tools": [
    {
      "name": "MyTool",
      "description": "Does something useful",
      "source": "./tools/my-tool.md"
    }
  ]
}
EOF
```

### Create a Tool

```bash
cat > .claude-plugin/tools/my-tool.md << 'EOF'
---
name: MyTool
description: A helpful tool
---

# MyTool

Use this tool to accomplish X by following these steps:

1. First step
2. Second step
3. Report results
EOF
```

## Architecture

The plugin system is built on these core components:

- **types.go**: Data structures for plugins, manifests, and registry entries
- **manifest.go**: Parsing and validation of `plugin.json`
- **loader.go**: Discovery and loading of plugins from filesystem
- **registry.go**: Central registry of all plugin extensions
- **tools.go**: Conversion of plugin tools to fantasy.AgentTool
- **hooks.go**: Integration with hook system
- **skills.go**: Integration with skill discovery system
- **mcp.go**: Stub for future MCP server integration
- **config.go**: Loading plugin configuration from crush.json
- **init.go**: Manager singleton and system initialization

## Limitations (Phase 3)

Not implemented yet:

- Marketplace with install/uninstall commands
- Binary plugins (only Markdown/JSON)
- Semantic versioning with cache per version
- CLI commands for plugin management
- Cross-plugin dependencies
- npm/git-remote plugins
- Hot reload without restart

## Status

✅ **Implemented:**
- Plugin discovery and loading
- Tool registration
- Hook integration
- Skill support
- Plugin configuration

📋 **Future Phases:**
- MCP server integration
- Plugin marketplace
- Binary plugin support
- Hot reload capability
