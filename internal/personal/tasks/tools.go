package tasks

import (
	"context"
	"fmt"
	"strings"

	"charm.land/fantasy"
	agenttools "github.com/charmbracelet/crush/internal/agent/tools"
)

const (
	taskCreateToolName = "task_create"
	taskGetToolName    = "task_get"
	taskUpdateToolName = "task_update"
	taskListToolName   = "task_list"
	taskDeleteToolName = "task_delete"
)

// BuildTools devuelve las tools de tareas.
func BuildTools() []fantasy.AgentTool {
	return []fantasy.AgentTool{
		newTaskCreateTool(),
		newTaskGetTool(),
		newTaskUpdateTool(),
		newTaskListTool(),
		newTaskDeleteTool(),
	}
}

type TaskCreateParams struct {
	Title       string `json:"title" description:"Título de la tarea"`
	Description string `json:"description,omitempty" description:"Descripción opcional"`
	Priority    string `json:"priority,omitempty" description:"Prioridad: low, medium o high"`
	ParentID    string `json:"parent_id,omitempty" description:"ID de la tarea padre"`
}

func newTaskCreateTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		taskCreateToolName,
		`Create a structured task for the current session.

Use this when you need durable task tracking with title, description, priority and optional parent task.`,
		func(ctx context.Context, params TaskCreateParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = call
			sessionID := agenttools.GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for creating tasks")
			}

			svc := GetService(sessionID)
			if svc == nil {
				return fantasy.ToolResponse{}, fmt.Errorf("tasks system is not initialized")
			}

			var parentPtr *string
			if parent := strings.TrimSpace(params.ParentID); parent != "" {
				parentPtr = &parent
			}

			priority := TaskPriority(strings.TrimSpace(params.Priority))
			if !priority.IsValid() {
				priority = TaskPriorityMedium
			}

			task, err := svc.Create(sessionID, strings.TrimSpace(params.Title), params.Description, priority, parentPtr)
			if err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf("Error creating task: %s", err.Error())), nil
			}

			return fantasy.NewTextResponse("Task created successfully.\n\n" + TaskToJSON(task)), nil
		},
	)
}

type TaskGetParams struct {
	ID string `json:"id" description:"ID de la tarea"`
}

func newTaskGetTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		taskGetToolName,
		`Get the full details of a task by its ID.`,
		func(ctx context.Context, params TaskGetParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = call
			sessionID := agenttools.GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for reading tasks")
			}

			svc := GetService(sessionID)
			if svc == nil {
				return fantasy.ToolResponse{}, fmt.Errorf("tasks system is not initialized")
			}

			task, err := svc.Get(sessionID, strings.TrimSpace(params.ID))
			if err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf("Error getting task: %s", err.Error())), nil
			}

			subtasks, err := svc.List(TaskFilter{SessionID: sessionID, ParentID: &task.ID})
			if err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf("Error listing subtasks: %s", err.Error())), nil
			}

			var response strings.Builder
			response.WriteString(TaskToJSON(task))
			if len(subtasks) > 0 {
				response.WriteString("\n\nSubtasks:\n")
				for _, subtask := range subtasks {
					response.WriteString(fmt.Sprintf("- [%s] %s (id: %s)\n", subtask.Status, subtask.Title, subtask.ID))
				}
			}
			return fantasy.NewTextResponse(response.String()), nil
		},
	)
}

type TaskUpdateParams struct {
	ID          string `json:"id" description:"ID de la tarea"`
	Title       string `json:"title,omitempty" description:"Nuevo título"`
	Description string `json:"description,omitempty" description:"Nueva descripción"`
	Status      string `json:"status,omitempty" description:"Nuevo estado: pending, in_progress o completed"`
	Priority    string `json:"priority,omitempty" description:"Nueva prioridad: low, medium o high"`
}

func newTaskUpdateTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		taskUpdateToolName,
		`Update an existing task. Only provided fields are modified.`,
		func(ctx context.Context, params TaskUpdateParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = call
			sessionID := agenttools.GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for updating tasks")
			}

			svc := GetService(sessionID)
			if svc == nil {
				return fantasy.ToolResponse{}, fmt.Errorf("tasks system is not initialized")
			}

			updates := TaskUpdate{}
			if trimmed := strings.TrimSpace(params.Title); trimmed != "" {
				updates.Title = &trimmed
			}
			if params.Description != "" {
				desc := params.Description
				updates.Description = &desc
			}
			if status := strings.TrimSpace(params.Status); status != "" {
				parsed := TaskStatus(status)
				updates.Status = &parsed
			}
			if priority := strings.TrimSpace(params.Priority); priority != "" {
				parsed := TaskPriority(priority)
				updates.Priority = &parsed
			}

			task, err := svc.Update(sessionID, strings.TrimSpace(params.ID), updates)
			if err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf("Error updating task: %s", err.Error())), nil
			}

			return fantasy.NewTextResponse("Task updated successfully.\n\n" + TaskToJSON(task)), nil
		},
	)
}

type TaskListParams struct {
	Status   string  `json:"status,omitempty" description:"Filtra por estado"`
	Priority string  `json:"priority,omitempty" description:"Filtra por prioridad"`
	ParentID *string `json:"parent_id,omitempty" description:"Filtra por tarea padre; usa cadena vacía para tareas raíz"`
}

func newTaskListTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		taskListToolName,
		`List tasks for the current session with optional filters.`,
		func(ctx context.Context, params TaskListParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = call
			sessionID := agenttools.GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for listing tasks")
			}

			svc := GetService(sessionID)
			if svc == nil {
				return fantasy.ToolResponse{}, fmt.Errorf("tasks system is not initialized")
			}

			filter := TaskFilter{SessionID: sessionID}
			if status := strings.TrimSpace(params.Status); status != "" {
				parsed := TaskStatus(status)
				filter.Status = &parsed
			}
			if priority := strings.TrimSpace(params.Priority); priority != "" {
				parsed := TaskPriority(priority)
				filter.Priority = &parsed
			}
			if params.ParentID != nil {
				parent := strings.TrimSpace(*params.ParentID)
				filter.ParentID = &parent
			}

			tasksList, err := svc.List(filter)
			if err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf("Error listing tasks: %s", err.Error())), nil
			}

			if len(tasksList) == 0 {
				return fantasy.NewTextResponse("No tasks found."), nil
			}

			var response strings.Builder
			response.WriteString(fmt.Sprintf("Found %d task(s).\n\n", len(tasksList)))
			for _, task := range tasksList {
				response.WriteString(fmt.Sprintf("- [%s] [%s] %s (id: %s)\n", task.Status, task.Priority, task.Title, task.ID))
				if task.Description != "" {
					response.WriteString("  " + task.Description + "\n")
				}
			}
			return fantasy.NewTextResponse(response.String()), nil
		},
	)
}

type TaskDeleteParams struct {
	ID string `json:"id" description:"ID de la tarea"`
}

func newTaskDeleteTool() fantasy.AgentTool {
	return fantasy.NewAgentTool(
		taskDeleteToolName,
		`Delete a task from the current session.

Use this when a task is no longer needed or was created by mistake.`,
		func(ctx context.Context, params TaskDeleteParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			_ = call
			sessionID := agenttools.GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("session ID is required for deleting tasks")
			}

			svc := GetService(sessionID)
			if svc == nil {
				return fantasy.ToolResponse{}, fmt.Errorf("tasks system is not initialized")
			}

			taskID := strings.TrimSpace(params.ID)
			task, err := svc.Get(sessionID, taskID)
			if err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf("Error deleting task: %s", err.Error())), nil
			}

			if err := svc.Delete(sessionID, taskID); err != nil {
				return fantasy.NewTextResponse(fmt.Sprintf("Error deleting task: %s", err.Error())), nil
			}

			return fantasy.NewTextResponse(
				"Task deleted successfully.\n\n" +
					fmt.Sprintf("Deleted task tree rooted at: %s (%s)\n", task.Title, task.ID),
			), nil
		},
	)
}
