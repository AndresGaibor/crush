-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS personal_cron_jobs;
