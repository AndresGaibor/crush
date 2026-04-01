package cron

import "time"

// CronScheduleKind define el tipo de programación.
type CronScheduleKind string

const (
	ScheduleCron      CronScheduleKind = "cron"
	ScheduleFixedRate CronScheduleKind = "fixed_rate"
	ScheduleOneTime   CronScheduleKind = "one_time"
)

// CronStatus representa el estado de un cron job.
type CronStatus string

const (
	CronEnabled  CronStatus = "enabled"
	CronDisabled CronStatus = "disabled"
	CronRunning  CronStatus = "running"
	CronError    CronStatus = "error"
)

// CronSchedule define el momento de ejecución.
type CronSchedule struct {
	Kind CronScheduleKind `json:"kind"`
	Expr string           `json:"expr"`
	TZ   string           `json:"tz,omitempty"`
}

// CronJob representa una tarea programada.
type CronJob struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Schedule   CronSchedule `json:"schedule"`
	Prompt     string       `json:"prompt"`
	Tools      []string     `json:"tools,omitempty"`
	Status     CronStatus   `json:"status"`
	Priority   int          `json:"priority,omitempty"`
	LastRunAt  *time.Time   `json:"last_run_at,omitempty"`
	NextRunAt  *time.Time   `json:"next_run_at,omitempty"`
	RunCount   int          `json:"run_count"`
	ErrorCount int          `json:"error_count"`
	LastError  string       `json:"last_error,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
	SessionID  string       `json:"session_id,omitempty"`
}

// CronConfig contiene la configuración del scheduler.
type CronConfig struct {
	MaxJobs           int `json:"max_jobs,omitempty"`
	MaxConcurrent     int `json:"max_concurrent,omitempty"`
	DefaultTimeoutSec int `json:"default_timeout_sec,omitempty"`
}

// DefaultCronConfig retorna la configuración por defecto.
func DefaultCronConfig() CronConfig {
	return CronConfig{
		MaxJobs:           50,
		MaxConcurrent:     2,
		DefaultTimeoutSec: 300,
	}
}
