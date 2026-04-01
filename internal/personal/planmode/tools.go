package planmode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/fantasy"
	agenttools "github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/google/uuid"
)

const (
	enterPlanModeToolName = "enter_plan_mode"
	exitPlanModeToolName  = "exit_plan_mode"
)

// BuildTools construye las tools expuestas por el subsistema plan mode.
func BuildTools() []fantasy.AgentTool {
	return []fantasy.AgentTool{
		newEnterPlanModeTool(),
		newExitPlanModeTool(),
	}
}

func newEnterPlanModeTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		enterPlanModeToolName,
		`Enter plan mode before making potentially destructive or multi-step changes.

Use this tool to register a structured plan and ask the user to review it.
If goal and steps are provided, the plan is stored immediately. Otherwise, the agent should present the plan in text and wait for approval.`,
		func(ctx context.Context, params PlanInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = call
			sessionID := agenttools.GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for plan mode")
			}

			sm := GetStateManager(sessionID)
			if sm == nil {
				return fantasy.ToolResponse{}, fmt.Errorf("plan mode is not initialized")
			}

			if err := sm.Enter(); err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf("Error entering plan mode: %s", err.Error())), nil
			}

			if params.Goal == "" || len(params.Steps) == 0 {
				return fantasy.NewTextResponse(renderPlanModeInstructions()), nil
			}

			plan := &Plan{
				ID:             uuid.NewString(),
				Goal:           params.Goal,
				Considerations: params.Considerations,
				AffectedFiles:  params.AffectedFiles,
				Steps:          make([]PlanStep, 0, len(params.Steps)),
			}
			for i, step := range params.Steps {
				if step.Number == 0 {
					step.Number = i + 1
				}
				plan.Steps = append(plan.Steps, PlanStep{
					Number:       step.Number,
					Description:  step.Description,
					Status:       StepPending,
					Dependencies: step.Dependencies,
				})
			}

			if err := sm.SetPlan(plan); err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf("Error setting plan: %s", err.Error())), nil
			}

			return fantasy.NewTextResponse(RenderPlan(plan) + "\n\n" + renderPlanModeReviewHint()), nil
		},
	)
}

// ExitPlanModeParams representa la decisión sobre el plan.
type ExitPlanModeParams struct {
	Decision      string `json:"decision" description:"Usa approve para continuar o reject para cancelar"`
	Reason        string `json:"reason,omitempty" description:"Razón de rechazo cuando decision sea reject"`
	Modifications string `json:"modifications,omitempty" description:"Cambios solicitados al plan en formato texto o JSON"`
}

func newExitPlanModeTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		exitPlanModeToolName,
		`Exit plan mode after the user has reviewed the proposed plan.

Use decision=approve to continue execution, or decision=reject to discard the plan and return to normal mode.`,
		func(ctx context.Context, params ExitPlanModeParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = call
			sessionID := agenttools.GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for plan mode")
			}

			sm := GetStateManager(sessionID)
			if sm == nil {
				return fantasy.ToolResponse{}, fmt.Errorf("plan mode is not initialized")
			}

			switch strings.ToLower(strings.TrimSpace(params.Decision)) {
			case "approve":
				if err := sm.Approve(); err != nil {
					return fantasy.NewTextResponse(fmt.Sprintf("Error approving plan: %s", err.Error())), nil
				}
				plan, err := sm.Exit()
				if err != nil {
					return fantasy.NewTextResponse(fmt.Sprintf("Error exiting plan mode: %s", err.Error())), nil
				}
				if plan == nil {
					return fantasy.NewTextResponse("Plan mode exited, but no plan was available."), nil
				}
				return fantasy.NewTextResponse(renderPlanApproved(plan)), nil

			case "reject":
				if err := sm.Reject(params.Reason); err != nil {
					return fantasy.NewTextResponse(fmt.Sprintf("Error rejecting plan: %s", err.Error())), nil
				}
				return fantasy.NewTextResponse(renderPlanRejected(params.Reason, params.Modifications)), nil

			default:
				return fantasy.NewTextResponse("Invalid decision. Use approve or reject."), nil
			}
		},
	)
}

// RenderPlan convierte el plan a texto legible.
func RenderPlan(plan *Plan) string {
	if plan == nil {
		return "No plan available."
	}

	var b strings.Builder
	b.WriteString("━━━ PLAN ━━━\n")
	b.WriteString(fmt.Sprintf("Goal: %s\n", plan.Goal))
	b.WriteString(fmt.Sprintf("Progress: %.0f%% (%d/%d steps)\n", plan.Progress()*100, completedCount(plan), len(plan.Steps)))
	b.WriteString(fmt.Sprintf("Created: %s\n\n", plan.CreatedAt.Format("2006-01-02 15:04:05")))

	b.WriteString("Steps:\n")
	for _, step := range plan.Steps {
		b.WriteString(fmt.Sprintf("- [%s] %d. %s", step.Status, step.Number, step.Description))
		if len(step.Dependencies) > 0 {
			b.WriteString(fmt.Sprintf(" (depends on %v)", step.Dependencies))
		}
		if step.Result != "" {
			b.WriteString("\n  Result: " + truncate(step.Result, 120))
		}
		if step.Error != "" {
			b.WriteString("\n  Error: " + truncate(step.Error, 120))
		}
		b.WriteString("\n")
	}

	if len(plan.Considerations) > 0 {
		b.WriteString("\nConsiderations:\n")
		for _, consideration := range plan.Considerations {
			b.WriteString("- " + consideration + "\n")
		}
	}

	if len(plan.AffectedFiles) > 0 {
		b.WriteString("\nAffected files:\n")
		for _, file := range plan.AffectedFiles {
			b.WriteString("- " + file + "\n")
		}
	}

	return strings.TrimSpace(b.String())
}

func completedCount(plan *Plan) int {
	done := 0
	for _, step := range plan.Steps {
		if step.Status == StepCompleted || step.Status == StepSkipped {
			done++
		}
	}
	return done
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	if limit <= 3 {
		return s[:limit]
	}
	return s[:limit-3] + "..."
}

// MarshalPlan devuelve el plan en JSON para facilitar pruebas o logs.
func MarshalPlan(plan *Plan) string {
	if plan == nil {
		return "{}"
	}
	b, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return string(b)
}

func renderPlanModeInstructions() string {
	return strings.Join([]string{
		"Plan mode activated.",
		"",
		"Next steps:",
		"- Generate a structured plan with a clear goal.",
		"- List steps in execution order.",
		"- Include any considerations or risks.",
		"- Mention affected files when relevant.",
		"- Wait for the user to review the plan before exiting plan mode.",
	}, "\n")
}

func renderPlanModeReviewHint() string {
	return "Review the plan with the user before calling exit_plan_mode."
}

func renderPlanApproved(plan *Plan) string {
	return strings.Join([]string{
		RenderPlan(plan),
		"",
		"Plan approved.",
		"Continue with the implementation one step at a time.",
	}, "\n")
}

func renderPlanRejected(reason, modifications string) string {
	lines := []string{
		"Plan rejected.",
		"Returned to normal mode.",
	}
	if reason != "" {
		lines = append(lines, "Reason: "+reason)
	}
	if modifications != "" {
		lines = append(lines, "Requested changes: "+modifications)
	}
	return strings.Join(lines, "\n")
}
