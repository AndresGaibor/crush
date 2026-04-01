package agent

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"charm.land/fantasy"

	"github.com/charmbracelet/crush/internal/agent/prompt"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/config"
	personalSubagents "github.com/charmbracelet/crush/internal/personal/subagents"
)

//go:embed templates/delegate_to_agent.md
var delegateToAgentDescription []byte

type delegateToAgentInput struct {
	Agent   string `json:"agent,omitempty" description:"Nombre del subagente a usar"`
	Task    string `json:"task" description:"Tarea concreta a delegar"`
	Context string `json:"context,omitempty" description:"Contexto adicional para el subagente"`
}

const DelegateToAgentToolName = "delegate_to_agent"

func (c *coordinator) delegateToAgentTool(ctx context.Context, parentAgent config.Agent) (fantasy.AgentTool, error) {
	return fantasy.NewAgentTool(
		DelegateToAgentToolName,
		string(delegateToAgentDescription),
		func(ctx context.Context, input delegateToAgentInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			task := strings.TrimSpace(input.Task)
			if task == "" {
				return fantasy.NewTextErrorResponse("task is required"), nil
			}

			subagent, err := c.resolveSubagent(input.Agent, task)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			sessionID := tools.GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, errors.New("session id missing from context")
			}

			agentMessageID := tools.GetMessageFromContext(ctx)
			if agentMessageID == "" {
				return fantasy.ToolResponse{}, errors.New("agent message id missing from context")
			}

			agentCfg := c.buildSubagentConfig(parentAgent, subagent)
			subagentPrompt, err := c.buildSubagentPrompt(ctx, subagent)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}

			agent, err := c.buildAgent(ctx, subagentPrompt, agentCfg, true)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}

			promptText := buildSubagentPromptText(task, input.Context)
			return c.runSubAgent(ctx, subAgentParams{
				Agent:          agent,
				SessionID:      sessionID,
				AgentMessageID: agentMessageID,
				ToolCallID:     call.ID,
				Prompt:         promptText,
				SessionTitle:   fmt.Sprintf("Subagent: %s", subagent.Name),
				SessionSetup: func(sessionID string) {
					c.permissions.AutoApproveSession(sessionID)
				},
			})
		},
	), nil
}

func (c *coordinator) resolveSubagent(name, task string) (*personalSubagents.Subagent, error) {
	name = strings.TrimSpace(name)
	task = strings.TrimSpace(task)

	if name != "" {
		if subagent, ok := personalSubagents.Get(name); ok {
			return subagent, nil
		}
		return nil, fmt.Errorf("subagent %q not found", name)
	}

	if subagent, ok := personalSubagents.Find(task); ok {
		return subagent, nil
	}

	return nil, fmt.Errorf("no subagent matches task %q", task)
}

func (c *coordinator) buildSubagentConfig(parentAgent config.Agent, subagent *personalSubagents.Subagent) config.Agent {
	allowedTools := parentAgent.AllowedTools
	filteredParentTools := make([]string, 0, len(parentAgent.AllowedTools))
	for _, tool := range parentAgent.AllowedTools {
		if tool == DelegateToAgentToolName {
			continue
		}
		filteredParentTools = append(filteredParentTools, tool)
	}
	allowedTools = filteredParentTools
	if len(subagent.Tools) > 0 {
		allowed := make([]string, 0, len(subagent.Tools))
		allowedSet := make(map[string]struct{}, len(parentAgent.AllowedTools))
		for _, tool := range parentAgent.AllowedTools {
			if tool == DelegateToAgentToolName {
				continue
			}
			allowedSet[tool] = struct{}{}
		}
		for _, tool := range subagent.Tools {
			if _, ok := allowedSet[tool]; ok {
				allowed = append(allowed, tool)
			}
		}
		allowedTools = allowed
	}
	model := parentAgent.Model
	switch subagent.Model {
	case "small":
		model = config.SelectedModelTypeSmall
	case "large":
		model = config.SelectedModelTypeLarge
	}

	return config.Agent{
		ID:           subagent.Name,
		Name:         subagent.Name,
		Description:  subagent.Description,
		Model:        model,
		AllowedTools: allowedTools,
		AllowedMCP:   parentAgent.AllowedMCP,
		ContextPaths: parentAgent.ContextPaths,
	}
}

func (c *coordinator) buildSubagentPrompt(ctx context.Context, subagent *personalSubagents.Subagent) (*prompt.Prompt, error) {
	basePrompt, err := coderPrompt(prompt.WithWorkingDir(c.cfg.WorkingDir()), prompt.WithSubagentMode())
	if err != nil {
		return nil, err
	}

	modelType := parentModelKey(subagent.Model)
	modelCfg, ok := c.cfg.Config().Models[modelType]
	if !ok {
		modelCfg = c.cfg.Config().Models[config.SelectedModelTypeLarge]
	}

	original, err := basePrompt.Build(ctx, modelCfg.Provider, modelCfg.Model, c.cfg)
	if err != nil {
		return nil, err
	}

	systemPrompt := original + "\n\n<subagent_definition>\n" + buildSubagentContext(subagent, "") + "\n</subagent_definition>"
	return prompt.NewPrompt("subagent", systemPrompt, prompt.WithWorkingDir(c.cfg.WorkingDir()), prompt.WithSubagentMode())
}

func parentModelKey(model string) config.SelectedModelType {
	switch model {
	case "small":
		return config.SelectedModelTypeSmall
	default:
		return config.SelectedModelTypeLarge
	}
}

func buildSubagentContext(subagent *personalSubagents.Subagent, extraContext string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("name: %s\n", subagent.Name))
	b.WriteString(fmt.Sprintf("description: %s\n", subagent.Description))
	b.WriteString(fmt.Sprintf("model: %s\n", subagent.Model))
	b.WriteString(fmt.Sprintf("visibility: %s\n", subagent.Visibility))
	if len(subagent.Tools) > 0 {
		b.WriteString(fmt.Sprintf("tools: %s\n", strings.Join(subagent.Tools, ", ")))
	}
	if strings.TrimSpace(subagent.Instructions) != "" {
		b.WriteString("instructions:\n")
		b.WriteString(strings.TrimSpace(subagent.Instructions))
		b.WriteString("\n")
	}
	if strings.TrimSpace(extraContext) != "" {
		b.WriteString("context:\n")
		b.WriteString(strings.TrimSpace(extraContext))
	}
	return strings.TrimSpace(b.String())
}

func buildSubagentPromptText(task, extraContext string) string {
	var b strings.Builder
	b.WriteString(task)
	if strings.TrimSpace(extraContext) != "" {
		b.WriteString("\n\nContexto adicional:\n")
		b.WriteString(strings.TrimSpace(extraContext))
	}
	return b.String()
}
