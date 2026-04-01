package planmode

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// Service persiste el estado del plan mode.
type Service struct {
	db *sql.DB
}

func newService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ensureSchema() error {
	_, err := s.db.Exec(`
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
`)
	return err
}

type persistedPlanMode struct {
	SessionID string
	Active    bool
	Approved  bool
	Plan      *Plan
	CreatedAt int64
	UpdatedAt int64
}

func (s *Service) load(sessionID string) (*persistedPlanMode, error) {
	row := s.db.QueryRow(`
SELECT session_id, active, approved, plan_json, created_at, updated_at
FROM personal_plan_modes
WHERE session_id = ?
`, sessionID)

	var item persistedPlanMode
	var active, approved int
	var planJSON sql.NullString
	if err := row.Scan(&item.SessionID, &active, &approved, &planJSON, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("loading plan mode state: %w", err)
	}

	item.Active = active != 0
	item.Approved = approved != 0
	if planJSON.Valid && planJSON.String != "" {
		var plan Plan
		if err := json.Unmarshal([]byte(planJSON.String), &plan); err != nil {
			return nil, fmt.Errorf("decoding persisted plan: %w", err)
		}
		plan.Normalize()
		item.Plan = &plan
	}

	return &item, nil
}

func (s *Service) save(sessionID string, active, approved bool, plan *Plan) error {
	now := time.Now().Unix()
	var planJSON sql.NullString
	if plan != nil {
		payload, err := json.Marshal(plan)
		if err != nil {
			return fmt.Errorf("encoding plan: %w", err)
		}
		planJSON = sql.NullString{String: string(payload), Valid: true}
	}

	_, err := s.db.Exec(`
INSERT INTO personal_plan_modes (session_id, active, approved, plan_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(session_id) DO UPDATE SET
	active = excluded.active,
	approved = excluded.approved,
	plan_json = excluded.plan_json,
	updated_at = excluded.updated_at
`, sessionID, boolToInt(active), boolToInt(approved), planJSON, now, now)
	if err != nil {
		return fmt.Errorf("saving plan mode state: %w", err)
	}

	slog.Debug("Plan mode state saved", "session_id", sessionID, "active", active, "approved", approved)
	return nil
}

func (s *Service) delete(sessionID string) error {
	_, err := s.db.Exec(`DELETE FROM personal_plan_modes WHERE session_id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("deleting plan mode state: %w", err)
	}
	slog.Debug("Plan mode state deleted", "session_id", sessionID)
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
