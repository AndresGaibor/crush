package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/fantasy"
	"github.com/charmbracelet/crush/internal/config"
)

// ConfigEditor expone la configuración necesaria para la herramienta.
type ConfigEditor interface {
	Config() *config.Config
	SetConfigField(scope config.Scope, key string, value any) error
	SetCompactMode(scope config.Scope, enabled bool) error
	SetTransparentBackground(scope config.Scope, enabled bool) error
}

type configToolInput struct {
	Action string `json:"action" description:"list, get o set"`
	Key    string `json:"key,omitempty" description:"Clave en notación dot para get/set"`
	Value  any    `json:"value,omitempty" description:"Nuevo valor para set"`
	Scope  string `json:"scope,omitempty" description:"global o workspace"`
}

// BuildConfigTool construye la herramienta ConfigTool.
func BuildConfigTool(editor ConfigEditor) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"config_tool",
		`Inspect and update a small allowlisted subset of Crush configuration.`,
		func(ctx context.Context, input configToolInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = ctx
			_ = call

			if editor == nil {
				return fantasy.NewTextErrorResponse("config editor is not available"), nil
			}

			switch strings.ToLower(strings.TrimSpace(input.Action)) {
			case "list":
				return listConfig(editor)
			case "get":
				return getConfigValue(editor, input.Key)
			case "set":
				return setConfigValue(editor, input.Scope, input.Key, input.Value)
			default:
				return fantasy.NewTextErrorResponse("action must be list, get, or set"), nil
			}
		},
	)
}

func listConfig(editor ConfigEditor) (fantasy.ToolResponse, error) {
	data, err := json.MarshalIndent(editor.Config(), "", "  ")
	if err != nil {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to serialize config: %s", err.Error())), nil
	}
	return fantasy.NewTextResponse(string(data)), nil
}

func getConfigValue(editor ConfigEditor, key string) (fantasy.ToolResponse, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return fantasy.NewTextErrorResponse("key is required"), nil
	}

	data, err := json.Marshal(editor.Config())
	if err != nil {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to serialize config: %s", err.Error())), nil
	}

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to parse config: %s", err.Error())), nil
	}

	value, ok := walkDotPath(root, key)
	if !ok {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("key %q not found", key)), nil
	}

	pretty, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fantasy.NewTextResponse(fmt.Sprintf("%v", value)), nil
	}
	return fantasy.NewTextResponse(string(pretty)), nil
}

func setConfigValue(editor ConfigEditor, scopeString, key string, value any) (fantasy.ToolResponse, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return fantasy.NewTextErrorResponse("key is required"), nil
	}

	scope := config.ScopeGlobal
	switch strings.ToLower(strings.TrimSpace(scopeString)) {
	case "", "global":
		scope = config.ScopeGlobal
	case "workspace":
		scope = config.ScopeWorkspace
	default:
		return fantasy.NewTextErrorResponse("scope must be global or workspace"), nil
	}

	switch key {
	case "options.tui.compact_mode":
		enabled, ok := toBool(value)
		if !ok {
			return fantasy.NewTextErrorResponse("compact_mode expects a boolean value"), nil
		}
		if err := editor.SetCompactMode(scope, enabled); err != nil {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to update config: %s", err.Error())), nil
		}
		return fantasy.NewTextResponse(fmt.Sprintf("updated %s = %v", key, enabled)), nil

	case "options.tui.transparent":
		enabled, ok := toBool(value)
		if !ok {
			return fantasy.NewTextErrorResponse("transparent expects a boolean value"), nil
		}
		if err := editor.SetTransparentBackground(scope, enabled); err != nil {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to update config: %s", err.Error())), nil
		}
		return fantasy.NewTextResponse(fmt.Sprintf("updated %s = %v", key, enabled)), nil

	case "options.disable_notifications":
		enabled, ok := toBool(value)
		if !ok {
			return fantasy.NewTextErrorResponse("disable_notifications expects a boolean value"), nil
		}
		if err := editor.SetConfigField(scope, key, enabled); err != nil {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to update config: %s", err.Error())), nil
		}
		if editor.Config() != nil && editor.Config().Options != nil {
			editor.Config().Options.DisableNotifications = enabled
		}
		return fantasy.NewTextResponse(fmt.Sprintf("updated %s = %v", key, enabled)), nil

	default:
		return fantasy.NewTextErrorResponse(fmt.Sprintf("key %q is read-only or not allowlisted", key)), nil
	}
}

func walkDotPath(root map[string]any, key string) (any, bool) {
	current := any(root)
	for _, part := range strings.Split(key, ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := lookupMapKey(obj, part)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func lookupMapKey(m map[string]any, key string) (any, bool) {
	if value, ok := m[key]; ok {
		return value, true
	}
	for k, value := range m {
		if strings.EqualFold(k, key) {
			return value, true
		}
	}
	return nil, false
}

func toBool(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
	case float64:
		return v != 0, true
	}
	return false, false
}
