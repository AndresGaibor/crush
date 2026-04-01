package cron

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Persistence maneja el almacenamiento SQLite de cron jobs.
type Persistence struct {
	db *sql.DB
}

// NewPersistence crea una nueva capa de persistencia.
func NewPersistence(db *sql.DB) (*Persistence, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	p := &Persistence{db: db}
	if err := p.ensureSchema(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Persistence) ensureSchema() error {
	_, err := p.db.Exec(`
CREATE TABLE IF NOT EXISTS personal_cron_jobs (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	schedule_kind TEXT NOT NULL DEFAULT 'cron',
	schedule_expr TEXT NOT NULL,
	schedule_tz TEXT NOT NULL DEFAULT 'UTC',
	prompt TEXT NOT NULL,
	tools TEXT NOT NULL DEFAULT '[]',
	status TEXT NOT NULL DEFAULT 'enabled',
	priority INTEGER NOT NULL DEFAULT 5,
	last_run_at TIMESTAMP,
	next_run_at TIMESTAMP,
	run_count INTEGER NOT NULL DEFAULT 0,
	error_count INTEGER NOT NULL DEFAULT 0,
	last_error TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	session_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_personal_cron_jobs_status ON personal_cron_jobs(status);
CREATE INDEX IF NOT EXISTS idx_personal_cron_jobs_next_run ON personal_cron_jobs(next_run_at);
`)
	if err != nil {
		return fmt.Errorf("creating cron schema: %w", err)
	}
	return nil
}

// Create inserta un job nuevo.
func (p *Persistence) Create(job *CronJob) error {
	toolsJSON, err := json.Marshal(job.Tools)
	if err != nil {
		return fmt.Errorf("serializing tools: %w", err)
	}

	_, err = p.db.Exec(`
INSERT INTO personal_cron_jobs (
	id, name, schedule_kind, schedule_expr, schedule_tz, prompt, tools, status,
	priority, last_run_at, next_run_at, run_count, error_count, last_error,
	created_at, updated_at, session_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, job.ID, job.Name, job.Schedule.Kind, job.Schedule.Expr, job.Schedule.TZ, job.Prompt, string(toolsJSON), job.Status,
		job.Priority, job.LastRunAt, job.NextRunAt, job.RunCount, job.ErrorCount, job.LastError, job.CreatedAt, job.UpdatedAt, job.SessionID)
	if err != nil {
		return fmt.Errorf("inserting cron job: %w", err)
	}
	return nil
}

// Update persiste un job existente.
func (p *Persistence) Update(job *CronJob) error {
	toolsJSON, err := json.Marshal(job.Tools)
	if err != nil {
		return fmt.Errorf("serializing tools: %w", err)
	}

	job.UpdatedAt = time.Now()
	_, err = p.db.Exec(`
UPDATE personal_cron_jobs SET
	name = ?, schedule_kind = ?, schedule_expr = ?, schedule_tz = ?, prompt = ?, tools = ?,
	status = ?, priority = ?, last_run_at = ?, next_run_at = ?, run_count = ?, error_count = ?,
	last_error = ?, updated_at = ?, session_id = ?
WHERE id = ?
`, job.Name, job.Schedule.Kind, job.Schedule.Expr, job.Schedule.TZ, job.Prompt, string(toolsJSON),
		job.Status, job.Priority, job.LastRunAt, job.NextRunAt, job.RunCount, job.ErrorCount,
		job.LastError, job.UpdatedAt, job.SessionID, job.ID)
	if err != nil {
		return fmt.Errorf("updating cron job: %w", err)
	}
	return nil
}

// Delete elimina un job.
func (p *Persistence) Delete(id string) error {
	result, err := p.db.Exec(`DELETE FROM personal_cron_jobs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting cron job: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("counting deleted rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("cron job %q not found", id)
	}
	return nil
}

// Get devuelve un job por ID.
func (p *Persistence) Get(id string) (*CronJob, error) {
	rows, err := p.List(nil)
	if err != nil {
		return nil, err
	}
	for _, job := range rows {
		if job.ID == id {
			return job, nil
		}
	}
	return nil, fmt.Errorf("cron job %q not found", id)
}

// List devuelve todos los jobs, con filtro opcional por estado.
func (p *Persistence) List(status *CronStatus) ([]*CronJob, error) {
	query := `
SELECT id, name, schedule_kind, schedule_expr, schedule_tz, prompt, tools, status,
	priority, last_run_at, next_run_at, run_count, error_count, last_error,
	created_at, updated_at, session_id
FROM personal_cron_jobs`
	var args []any
	if status != nil {
		query += ` WHERE status = ?`
		args = append(args, string(*status))
	}
	query += ` ORDER BY created_at ASC`

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying cron jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]*CronJob, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating cron jobs: %w", err)
	}
	return jobs, nil
}

// Count devuelve la cantidad de jobs persistidos.
func (p *Persistence) Count() (int, error) {
	var count int
	if err := p.db.QueryRow(`SELECT COUNT(*) FROM personal_cron_jobs`).Scan(&count); err != nil {
		return 0, fmt.Errorf("counting cron jobs: %w", err)
	}
	return count, nil
}

func scanJob(rows *sql.Rows) (*CronJob, error) {
	job := &CronJob{}
	var toolsJSON string
	var lastRun, nextRun sql.NullTime
	if err := rows.Scan(
		&job.ID, &job.Name, &job.Schedule.Kind, &job.Schedule.Expr, &job.Schedule.TZ, &job.Prompt, &toolsJSON, &job.Status,
		&job.Priority, &lastRun, &nextRun, &job.RunCount, &job.ErrorCount, &job.LastError,
		&job.CreatedAt, &job.UpdatedAt, &job.SessionID,
	); err != nil {
		return nil, fmt.Errorf("scanning cron job: %w", err)
	}

	if toolsJSON != "" {
		if err := json.Unmarshal([]byte(toolsJSON), &job.Tools); err != nil {
			return nil, fmt.Errorf("decoding cron tools: %w", err)
		}
	}
	if lastRun.Valid {
		job.LastRunAt = &lastRun.Time
	}
	if nextRun.Valid {
		job.NextRunAt = &nextRun.Time
	}
	job.Schedule.Kind = CronScheduleKind(strings.TrimSpace(string(job.Schedule.Kind)))
	job.Status = CronStatus(strings.TrimSpace(string(job.Status)))
	return job, nil
}
