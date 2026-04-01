-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS personal_plan_modes (
    session_id TEXT PRIMARY KEY,
    active INTEGER NOT NULL DEFAULT 0,
    approved INTEGER NOT NULL DEFAULT 0,
    plan_json TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_personal_plan_modes_active ON personal_plan_modes(active);
CREATE INDEX IF NOT EXISTS idx_personal_plan_modes_approved ON personal_plan_modes(approved);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS personal_plan_modes;
-- +goose StatementEnd
