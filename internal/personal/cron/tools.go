package cron

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/fantasy"
)

type cronCreateInput struct {
	Name         string   `json:"name" description:"Nombre descriptivo del cron job"`
	ScheduleKind string   `json:"schedule_kind" description:"cron, fixed_rate o one_time"`
	ScheduleExpr string   `json:"schedule_expr" description:"Expresión de cron, segundos o timestamp"`
	Prompt       string   `json:"prompt" description:"Prompt que se ejecutará"`
	Tools        []string `json:"tools,omitempty" description:"Herramientas sugeridas para la ejecución"`
	Priority     int      `json:"priority,omitempty" description:"Prioridad del job"`
	TZ           string   `json:"tz,omitempty" description:"Zona horaria"`
	SessionID    string   `json:"session_id,omitempty" description:"Sesión destino"`
}

type cronDeleteInput struct {
	ID string `json:"id" description:"ID del cron job"`
}

type cronListInput struct {
	Status string `json:"status,omitempty" description:"enabled, disabled, running o error"`
}

// BuildTools construye las herramientas públicas del subsistema cron.
func BuildTools() []fantasy.AgentTool {
	return []fantasy.AgentTool{
		BuildCronCreateTool(),
		BuildCronDeleteTool(),
		BuildCronListTool(),
	}
}

// BuildCronCreateTool construye la herramienta CronCreate.
func BuildCronCreateTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"cron_create",
		`Create a new scheduled job with a cron expression, fixed interval, or one-time timestamp.`,
		func(ctx context.Context, input cronCreateInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = ctx
			_ = call

			scheduler := GetScheduler()
			if scheduler == nil {
				return fantasy.NewTextErrorResponse("cron scheduler is not available"), nil
			}

			name := strings.TrimSpace(input.Name)
			prompt := strings.TrimSpace(input.Prompt)
			kind := CronScheduleKind(strings.TrimSpace(input.ScheduleKind))
			expr := strings.TrimSpace(input.ScheduleExpr)
			if name == "" || prompt == "" || kind == "" || expr == "" {
				return fantasy.NewTextErrorResponse("name, schedule_kind, schedule_expr, and prompt are required"), nil
			}

			job := &CronJob{
				ID:        generateCronJobID(),
				Name:      name,
				Schedule:  CronSchedule{Kind: kind, Expr: expr, TZ: strings.TrimSpace(input.TZ)},
				Prompt:    prompt,
				Tools:     append([]string(nil), input.Tools...),
				Status:    CronEnabled,
				Priority:  input.Priority,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				SessionID: strings.TrimSpace(input.SessionID),
			}
			if job.Priority == 0 {
				job.Priority = 5
			}
			if job.Schedule.TZ == "" {
				job.Schedule.TZ = "UTC"
			}

			if err := scheduler.AddJob(job); err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to create cron job: %s", err.Error())), nil
			}

			var b strings.Builder
			b.WriteString("Cron job created successfully.\n")
			b.WriteString(fmt.Sprintf("- ID: %s\n", job.ID))
			b.WriteString(fmt.Sprintf("- Name: %s\n", job.Name))
			b.WriteString(fmt.Sprintf("- Schedule: %s (%s)\n", job.Schedule.Kind, job.Schedule.Expr))
			b.WriteString(fmt.Sprintf("- Status: %s\n", job.Status))
			if len(job.Tools) > 0 {
				b.WriteString(fmt.Sprintf("- Tools: %s\n", strings.Join(job.Tools, ", ")))
			}
			return fantasy.NewTextResponse(b.String()), nil
		},
	)
}

// BuildCronDeleteTool construye la herramienta CronDelete.
func BuildCronDeleteTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"cron_delete",
		`Delete a scheduled cron job.`,
		func(ctx context.Context, input cronDeleteInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = ctx
			_ = call

			scheduler := GetScheduler()
			if scheduler == nil {
				return fantasy.NewTextErrorResponse("cron scheduler is not available"), nil
			}

			id := strings.TrimSpace(input.ID)
			if id == "" {
				return fantasy.NewTextErrorResponse("id is required"), nil
			}

			if err := scheduler.RemoveJob(id); err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("failed to delete cron job: %s", err.Error())), nil
			}
			return fantasy.NewTextResponse(fmt.Sprintf("Cron job %s deleted successfully.", id)), nil
		},
	)
}

// BuildCronListTool construye la herramienta CronList.
func BuildCronListTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		"cron_list",
		`List scheduled cron jobs.`,
		func(ctx context.Context, input cronListInput, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = ctx
			_ = call

			scheduler := GetScheduler()
			if scheduler == nil {
				return fantasy.NewTextErrorResponse("cron scheduler is not available"), nil
			}

			var filter *CronStatus
			if status := strings.TrimSpace(input.Status); status != "" {
				parsed := CronStatus(status)
				filter = &parsed
			}

			jobs := scheduler.ListJobs(filter)
			if len(jobs) == 0 {
				return fantasy.NewTextResponse("No cron jobs found."), nil
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("Found %d cron job(s):\n\n", len(jobs)))
			for _, job := range jobs {
				b.WriteString(fmt.Sprintf("- %s [%s] (%s)\n", job.Name, job.Status, job.ID))
				b.WriteString(fmt.Sprintf("  Schedule: %s (%s)\n", job.Schedule.Kind, job.Schedule.Expr))
				if job.NextRunAt != nil {
					b.WriteString(fmt.Sprintf("  Next run: %s\n", job.NextRunAt.Format(time.RFC3339)))
				}
				if job.LastRunAt != nil {
					b.WriteString(fmt.Sprintf("  Last run: %s\n", job.LastRunAt.Format(time.RFC3339)))
				}
				if job.LastError != "" {
					b.WriteString(fmt.Sprintf("  Last error: %s\n", job.LastError))
				}
				if len(job.Tools) > 0 {
					b.WriteString(fmt.Sprintf("  Tools: %s\n", strings.Join(job.Tools, ", ")))
				}
				b.WriteString("\n")
			}

			return fantasy.NewTextResponse(b.String()), nil
		},
	)
}

func generateCronJobID() string {
	return fmt.Sprintf("cron_%d", time.Now().UnixNano())
}
