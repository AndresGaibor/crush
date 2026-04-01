package tasks

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"charm.land/fantasy"
	_ "modernc.org/sqlite"

	agenttools "github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildToolsNames(t *testing.T) {
	t.Parallel()

	tools := BuildTools()
	require.Len(t, tools, 5)

	assert.Equal(t, taskCreateToolName, tools[0].Info().Name)
	assert.Equal(t, taskGetToolName, tools[1].Info().Name)
	assert.Equal(t, taskUpdateToolName, tools[2].Info().Name)
	assert.Equal(t, taskListToolName, tools[3].Info().Name)
	assert.Equal(t, taskDeleteToolName, tools[4].Info().Name)
}

func TestInitAndGetServiceRegistry(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	db := openTestDB(t)
	require.NoError(t, Init(db))

	first := GetService("session-a")
	require.NotNil(t, first)
	second := GetService("session-a")
	require.Same(t, first, second)

	assert.Nil(t, GetService(""))
}

func TestServiceCRUD(t *testing.T) {
	db := openTestDB(t)
	svc := newService(db)
	require.NoError(t, svc.ensureSchema())

	parent, err := svc.Create("session-a", "Planificar el trabajo", "Tarea principal", TaskPriorityHigh, nil)
	require.NoError(t, err)
	require.NotEmpty(t, parent.ID)
	assert.Equal(t, TaskPriorityHigh, parent.Priority)

	fallback, err := svc.Create("session-a", "Tarea sin prioridad válida", "", TaskPriority("invalid"), nil)
	require.NoError(t, err)
	assert.Equal(t, TaskPriorityMedium, fallback.Priority)

	child, err := svc.Create("session-a", "Implementar subtarea", "Detalle", TaskPriorityLow, &parent.ID)
	require.NoError(t, err)
	require.NotNil(t, child.ParentID)
	assert.Equal(t, parent.ID, *child.ParentID)

	gotParent, err := svc.Get("session-a", parent.ID)
	require.NoError(t, err)
	assert.Equal(t, "Planificar el trabajo", gotParent.Title)

	allTasks, err := svc.List(TaskFilter{SessionID: "session-a"})
	require.NoError(t, err)
	require.Len(t, allTasks, 3)

	rootOnly := ""
	rootTasks, err := svc.List(TaskFilter{SessionID: "session-a", ParentID: &rootOnly})
	require.NoError(t, err)
	require.Len(t, rootTasks, 2)

	completed := TaskStatusCompleted
	updated, err := svc.Update("session-a", child.ID, TaskUpdate{Status: &completed})
	require.NoError(t, err)
	assert.Equal(t, TaskStatusCompleted, updated.Status)
	require.NotNil(t, updated.CompletedAt)

	gotChild, err := svc.Get("session-a", child.ID)
	require.NoError(t, err)
	require.NotNil(t, gotChild.CompletedAt)

	status := TaskStatusCompleted
	completedTasks, err := svc.List(TaskFilter{SessionID: "session-a", Status: &status})
	require.NoError(t, err)
	require.Len(t, completedTasks, 1)
	assert.Equal(t, child.ID, completedTasks[0].ID)

	require.NoError(t, svc.Delete("session-a", fallback.ID))

	_, err = svc.Get("session-a", fallback.ID)
	require.Error(t, err)
}

func TestTaskDeleteToolDeletesTask(t *testing.T) {
	db := openTestDB(t)
	Reset()
	t.Cleanup(Reset)
	require.NoError(t, Init(db))

	svc := GetService("session-a")
	require.NotNil(t, svc)

	task, err := svc.Create("session-a", "Tarea a borrar", "", TaskPriorityMedium, nil)
	require.NoError(t, err)

	tool := newTaskDeleteTool()
	ctx := t.Context()
	ctx = context.WithValue(ctx, agenttools.SessionIDContextKey, "session-a")

	resp, err := tool.Run(ctx, fantasy.ToolCall{
		ID:    "call-delete",
		Name:  taskDeleteToolName,
		Input: `{"id":"` + task.ID + `"}`,
	})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	assert.Contains(t, resp.Content, "Task deleted successfully")

	_, err = svc.Get("session-a", task.ID)
	require.Error(t, err)
}

func TestTaskDeleteRemovesSubtree(t *testing.T) {
	db := openTestDB(t)
	svc := newService(db)
	require.NoError(t, svc.ensureSchema())

	parent, err := svc.Create("session-a", "Padre", "", TaskPriorityMedium, nil)
	require.NoError(t, err)
	child, err := svc.Create("session-a", "Hija", "", TaskPriorityMedium, &parent.ID)
	require.NoError(t, err)
	_, err = svc.Create("session-a", "Nieta", "", TaskPriorityMedium, &child.ID)
	require.NoError(t, err)

	require.NoError(t, svc.Delete("session-a", parent.ID))

	_, err = svc.Get("session-a", parent.ID)
	require.Error(t, err)
	_, err = svc.Get("session-a", child.ID)
	require.Error(t, err)
}

func TestCreateRejectsParentFromOtherSession(t *testing.T) {
	db := openTestDB(t)
	svc := newService(db)
	require.NoError(t, svc.ensureSchema())

	parent, err := svc.Create("session-a", "Padre", "", TaskPriorityMedium, nil)
	require.NoError(t, err)

	_, err = svc.Create("session-b", "Hija", "", TaskPriorityMedium, &parent.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parent")
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), fmt.Sprintf("%s.db", t.Name()))
	db, err := sql.Open("sqlite", "file:"+dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
