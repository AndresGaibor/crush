package plugins

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()

	registry.Register(RegistryEntry{
		PluginID: "test@project",
		Type:     "tool",
		Name:     "MyTool",
		Data:     ToolDecl{Name: "MyTool"},
	})

	tools := registry.GetTools()
	assert.Len(t, tools, 1)
	assert.Equal(t, "MyTool", tools[0].Name)
	assert.Equal(t, PluginID("test@project"), tools[0].PluginID)
}

func TestRegistry_Unregister(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()

	registry.Register(RegistryEntry{PluginID: "p1@project", Type: "tool", Name: "T1"})
	registry.Register(RegistryEntry{PluginID: "p2@project", Type: "tool", Name: "T2"})
	assert.Len(t, registry.GetTools(), 2)

	registry.Unregister("p1@project")
	assert.Len(t, registry.GetTools(), 1)
	assert.Equal(t, "T2", registry.GetTools()[0].Name)
}

func TestRegistry_PopulateFromPlugins(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()

	plugins := []*Plugin{
		{
			ID:     "test@project",
			Name:   "test",
			Status: StatusEnabled,
			Manifest: &Manifest{
				Tools: []ToolDecl{
					{Name: "Tool1", Source: "./t1.md"},
					{Name: "Tool2", Source: "./t2.md"},
				},
				Skills: []string{"./skills/deploy"},
				Hooks:  json.RawMessage(`{"hooks":{"PostToolUse":[{"matcher":"*","hooks":[{"type":"command","command":"echo hi"}]}]}}`),
			},
		},
	}

	registry.PopulateFromPlugins(plugins)

	assert.Len(t, registry.GetTools(), 2)
	assert.Len(t, registry.GetSkills(), 1)
	assert.Len(t, registry.GetHooks(), 1)

	counts := registry.Count()
	assert.Equal(t, 2, counts["tool"])
	assert.Equal(t, 1, counts["skill"])
	assert.Equal(t, 1, counts["hook"])
}

func TestRegistry_SkipDisabledPlugins(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()

	plugins := []*Plugin{
		{ID: "enabled@project", Name: "enabled", Status: StatusEnabled,
			Manifest: &Manifest{Tools: []ToolDecl{{Name: "T1", Source: "./t.md"}}}},
		{ID: "disabled@project", Name: "disabled", Status: StatusDisabled,
			Manifest: &Manifest{Tools: []ToolDecl{{Name: "T2", Source: "./t.md"}}}},
		{ID: "error@project", Name: "error", Status: StatusError,
			Manifest: nil},
	}

	registry.PopulateFromPlugins(plugins)
	assert.Len(t, registry.GetTools(), 1)
	assert.Equal(t, "enabled@project:T1", registry.GetTools()[0].Name)
}

func TestRegistry_String(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	registry.Register(RegistryEntry{Type: "tool", Name: "T1"})
	registry.Register(RegistryEntry{Type: "hook", Name: "H1"})
	registry.Register(RegistryEntry{Type: "tool", Name: "T2"})

	s := registry.String()
	assert.Contains(t, s, "hook:1")
	assert.Contains(t, s, "tool:2")
}

func TestRegistry_PopulateFromPlugins_DetectsToolCollision(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()

	plugins := []*Plugin{
		{
			ID:     "first@project",
			Name:   "first",
			Status: StatusEnabled,
			Manifest: &Manifest{
				Tools: []ToolDecl{{Name: "SharedTool", Source: "./t1.md"}},
			},
		},
		{
			ID:     "second@project",
			Name:   "second",
			Status: StatusEnabled,
			Manifest: &Manifest{
				Tools: []ToolDecl{{Name: "SharedTool", Source: "./t2.md"}},
			},
		},
	}

	registry.PopulateFromPlugins(plugins)
	assert.Len(t, registry.GetTools(), 1)
	assert.Len(t, registry.Collisions(), 1)
	assert.Equal(t, "tool", registry.Collisions()[0].Type)
	assert.Equal(t, PluginID("first@project"), registry.Collisions()[0].WinnerPluginID)
	assert.Equal(t, PluginID("second@project"), registry.Collisions()[0].LoserPluginID)
}
