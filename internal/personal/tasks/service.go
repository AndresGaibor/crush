package tasks

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const defaultMaxTasksPerSession = 100

// Service ejecuta operaciones CRUD sobre la tabla de tareas.
type Service struct {
	db *sql.DB
}

// newService crea el servicio sobre la conexión compartida.
func newService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ensureSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS personal_tasks (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'in_progress', 'completed')),
	priority TEXT NOT NULL DEFAULT 'medium' CHECK(priority IN ('low', 'medium', 'high')),
	parent_id TEXT,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	completed_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_personal_tasks_session_id ON personal_tasks(session_id);
CREATE INDEX IF NOT EXISTS idx_personal_tasks_status ON personal_tasks(status);
CREATE INDEX IF NOT EXISTS idx_personal_tasks_priority ON personal_tasks(priority);
CREATE INDEX IF NOT EXISTS idx_personal_tasks_parent_id ON personal_tasks(parent_id);
`)
	return err
}

// Create crea una tarea nueva.
func (s *Service) Create(sessionID, title, description string, priority TaskPriority, parentID *string) (*Task, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if !priority.IsValid() {
		priority = TaskPriorityMedium
	}

	count, err := s.countBySession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("counting tasks: %w", err)
	}
	if count >= defaultMaxTasksPerSession {
		return nil, fmt.Errorf("maximum tasks per session reached (%d)", defaultMaxTasksPerSession)
	}

	if parentID != nil && *parentID != "" {
		parent, err := s.Get(sessionID, *parentID)
		if err != nil {
			return nil, fmt.Errorf("checking parent task: %w", err)
		}
		if parent.SessionID != sessionID {
			return nil, fmt.Errorf("parent task must belong to the same session")
		}
	}

	now := time.Now().Unix()
	task := &Task{
		ID:          uuid.NewString(),
		SessionID:   sessionID,
		Title:       title,
		Description: description,
		Status:      TaskStatusPending,
		Priority:    priority,
		ParentID:    parentID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = s.db.Exec(`
INSERT INTO personal_tasks (id, session_id, title, description, status, priority, parent_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`, task.ID, task.SessionID, task.Title, task.Description, task.Status, task.Priority, task.ParentID, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting task: %w", err)
	}

	slog.Debug("Task created", "id", task.ID, "session_id", task.SessionID)
	return task, nil
}

// Get recupera una tarea por ID dentro de la sesión actual.
func (s *Service) Get(sessionID, id string) (*Task, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if id == "" {
		return nil, fmt.Errorf("task id is required")
	}

	task, err := scanTask(s.db.QueryRow(`
SELECT id, session_id, title, description, status, priority, parent_id, created_at, updated_at, completed_at
FROM personal_tasks
WHERE id = ? AND session_id = ?
`, id, sessionID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task %q not found", id)
		}
		return nil, fmt.Errorf("querying task: %w", err)
	}

	return task, nil
}

// Update modifica una tarea existente.
func (s *Service) Update(sessionID, id string, updates TaskUpdate) (*Task, error) {
	task, err := s.Get(sessionID, id)
	if err != nil {
		return nil, err
	}

	if updates.Title != nil && *updates.Title != "" {
		task.Title = *updates.Title
	}
	if updates.Description != nil {
		task.Description = *updates.Description
	}
	if updates.Priority != nil {
		if !updates.Priority.IsValid() {
			return nil, fmt.Errorf("invalid priority: %q", *updates.Priority)
		}
		task.Priority = *updates.Priority
	}
	if updates.Status != nil {
		if !updates.Status.IsValid() {
			return nil, fmt.Errorf("invalid status: %q", *updates.Status)
		}
		task.Status = *updates.Status
		if task.Status == TaskStatusCompleted {
			now := time.Now().Unix()
			task.CompletedAt = &now
		} else {
			task.CompletedAt = nil
		}
	}

	task.UpdatedAt = time.Now().Unix()
	_, err = s.db.Exec(`
UPDATE personal_tasks
SET title = ?, description = ?, status = ?, priority = ?, parent_id = ?, updated_at = ?, completed_at = ?
WHERE id = ? AND session_id = ?
`, task.Title, task.Description, task.Status, task.Priority, task.ParentID, task.UpdatedAt, task.CompletedAt, task.ID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("updating task: %w", err)
	}

	slog.Debug("Task updated", "id", task.ID, "session_id", task.SessionID)
	return task, nil
}

// List devuelve tareas filtradas por sesión y atributos opcionales.
func (s *Service) List(filter TaskFilter) ([]*Task, error) {
	if filter.SessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	query := `
SELECT id, session_id, title, description, status, priority, parent_id, created_at, updated_at, completed_at
FROM personal_tasks
WHERE session_id = ?`
	args := []any{filter.SessionID}

	if filter.Status != nil {
		if !filter.Status.IsValid() {
			return nil, fmt.Errorf("invalid status: %q", *filter.Status)
		}
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}
	if filter.Priority != nil {
		if !filter.Priority.IsValid() {
			return nil, fmt.Errorf("invalid priority: %q", *filter.Priority)
		}
		query += " AND priority = ?"
		args = append(args, *filter.Priority)
	}
	if filter.ParentID != nil {
		if *filter.ParentID == "" {
			query += " AND parent_id IS NULL"
		} else {
			query += " AND parent_id = ?"
			args = append(args, *filter.ParentID)
		}
	}

	query += " ORDER BY created_at ASC"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning task row: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating task rows: %w", err)
	}

	return tasks, nil
}

// Delete borra una tarea de la sesión actual.
func (s *Service) Delete(sessionID, id string) error {
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if id == "" {
		return fmt.Errorf("task id is required")
	}

	result, err := s.db.Exec(`
WITH RECURSIVE subtree(id) AS (
	SELECT id
	FROM personal_tasks
	WHERE id = ? AND session_id = ?
	UNION ALL
	SELECT t.id
	FROM personal_tasks t
	JOIN subtree s ON t.parent_id = s.id
	WHERE t.session_id = ?
)
DELETE FROM personal_tasks
WHERE id IN (SELECT id FROM subtree)
`, id, sessionID, sessionID)
	if err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("task %q not found", id)
	}

	slog.Debug("Task deleted", "id", id, "session_id", sessionID)
	return nil
}

func (s *Service) countBySession(sessionID string) (int, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM personal_tasks WHERE session_id = ?`, sessionID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func scanTask(row interface{ Scan(dest ...any) error }) (*Task, error) {
	task := &Task{}
	var parentID sql.NullString
	var completedAt sql.NullInt64
	err := row.Scan(
		&task.ID,
		&task.SessionID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.Priority,
		&parentID,
		&task.CreatedAt,
		&task.UpdatedAt,
		&completedAt,
	)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		task.ParentID = &parentID.String
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Int64
	}
	return task, nil
}

// TaskToJSON devuelve la tarea serializada para la respuesta de la tool.
func TaskToJSON(task *Task) string {
	if task == nil {
		return "{}"
	}
	b, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return string(b)
}

// TasksToJSON devuelve la lista de tareas serializada.
func TasksToJSON(tasks []*Task) string {
	b, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return string(b)
}
