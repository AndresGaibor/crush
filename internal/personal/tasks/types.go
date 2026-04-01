package tasks

// TaskStatus representa el estado de una tarea.
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
)

// IsValid valida el estado.
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusPending, TaskStatusInProgress, TaskStatusCompleted:
		return true
	default:
		return false
	}
}

// TaskPriority representa la prioridad de una tarea.
type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

// IsValid valida la prioridad.
func (p TaskPriority) IsValid() bool {
	switch p {
	case TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh:
		return true
	default:
		return false
	}
}

// Task representa una tarea persistida.
type Task struct {
	ID          string       `json:"id"`
	SessionID   string       `json:"session_id"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Status      TaskStatus   `json:"status"`
	Priority    TaskPriority `json:"priority"`
	ParentID    *string      `json:"parent_id,omitempty"`
	CreatedAt   int64        `json:"created_at"`
	UpdatedAt   int64        `json:"updated_at"`
	CompletedAt *int64       `json:"completed_at,omitempty"`
}

// TaskFilter define los filtros de listado.
type TaskFilter struct {
	SessionID string
	Status    *TaskStatus
	Priority  *TaskPriority
	ParentID  *string
}

// TaskUpdate representa actualizaciones parciales.
type TaskUpdate struct {
	Title       *string
	Description *string
	Status      *TaskStatus
	Priority    *TaskPriority
}
