package plugins

import (
	"encoding/json"
	"slices"

	appconfig "github.com/charmbracelet/crush/internal/config"
	personalhooks "github.com/charmbracelet/crush/internal/personal/hooks"
)

// ApplyToConfig integra los plugins cargados con la configuración de Crush.
// Debe llamarse una vez después de Init y antes de inicializar MCP o construir
// prompts/agentes.
func (m *Manager) ApplyToConfig(cfg *appconfig.Config, hookMgr *personalhooks.Manager) {
	if m == nil || cfg == nil {
		return
	}

	if cfg.Options == nil {
		cfg.Options = &appconfig.Options{}
	}

	// Hooks: conservar los del usuario y sumar los de plugins.
	if hookMgr != nil {
		pluginHooks := personalhooks.HookConfigMap{}
		for _, entry := range m.Registry.GetHooks() {
			hooksData, ok := entry.Data.(json.RawMessage)
			if !ok {
				continue
			}

			loaded, err := personalhooks.LoadFromBytes(hooksData)
			if err != nil {
				continue
			}
			pluginHooks = personalhooks.Merge(pluginHooks, loaded)
		}

		if len(pluginHooks) > 0 {
			merged := personalhooks.Merge(hookMgr.Config(), pluginHooks)
			hookMgr.LoadConfig(merged)
		}
	}

	// Skills: agregar paths únicos.
	for _, skillPath := range GetPluginSkillPaths(m.Registry.GetSkills()) {
		if !slices.Contains(cfg.Options.SkillsPaths, skillPath) {
			cfg.Options.SkillsPaths = append(cfg.Options.SkillsPaths, skillPath)
		}
	}

	// Tools: habilitar los tools de plugins para los agentes existentes.
	pluginTools := m.Registry.GetTools()
	if len(pluginTools) > 0 && cfg.Agents != nil {
		for _, agent := range cfg.Agents {
			agent.AllowedTools = appendUniqueTools(agent.AllowedTools, pluginTools)
			cfg.Agents[agent.ID] = agent
		}
	}

	// MCP: agregar servidores sin pisar los del usuario.
	if len(cfg.MCP) > 0 || len(m.Registry.GetMCPServers()) > 0 {
		cfg.MCP = MergePluginMCPIntoConfig(cfg.MCP, GetPluginMCPServers(m.Registry.GetMCPServers()))
	}
}

// ReloadAndApply recarga los plugins y reaplica sus integraciones a la config.
func (m *Manager) ReloadAndApply(cfg *appconfig.Config, hookMgr *personalhooks.Manager) error {
	if err := m.Reload(); err != nil {
		return err
	}
	m.ApplyToConfig(cfg, hookMgr)
	return nil
}

func appendUniqueTools(existing []string, entries []RegistryEntry) []string {
	out := append([]string(nil), existing...)
	for _, entry := range entries {
		if entry.Type != "tool" {
			continue
		}
		decl, ok := entry.Data.(ToolDecl)
		if !ok {
			continue
		}
		if !slices.Contains(out, decl.Name) {
			out = append(out, decl.Name)
		}
	}
	return out
}
