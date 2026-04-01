package cron

import (
	"database/sql"
	"log/slog"
	"sync"
)

var (
	instance     *Scheduler
	instanceOnce sync.Once
)

// Init inicializa el subsistema de cron.
func Init(db *sql.DB, cfg CronConfig) error {
	var initErr error
	instanceOnce.Do(func() {
		persistence, err := NewPersistence(db)
		if err != nil {
			initErr = err
			return
		}

		instance = NewScheduler(persistence, cfg, nil)
		if err := instance.LoadJobs(); err != nil {
			slog.Warn("Failed to load cron jobs", "error", err)
		}
		slog.Info("Cron subsystem initialized", "max_jobs", cfg.MaxJobs, "default_timeout_sec", cfg.DefaultTimeoutSec)
	})
	return initErr
}

// SetRunner configura la función que ejecuta los jobs.
func SetRunner(runner Runner) {
	if instance == nil {
		return
	}
	instance.SetRunner(runner)
}

// GetScheduler devuelve el scheduler global.
func GetScheduler() *Scheduler {
	return instance
}

// IsInitialized informa si el subsistema está listo.
func IsInitialized() bool {
	return instance != nil
}

// Shutdown detiene el scheduler global.
func Shutdown() {
	if instance != nil {
		instance.Stop()
	}
}

// Reset reinicia el singleton para tests.
func Reset() {
	instance = nil
	instanceOnce = sync.Once{}
}
