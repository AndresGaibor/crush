package planmode

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// StateManager mantiene el estado del plan mode por sesión.
type StateManager struct {
	mu        sync.RWMutex
	sessionID string
	service   *Service
	active    bool
	approved  bool
	plan      *Plan
}

// NewStateManager crea un gestor vacío.
func NewStateManager() *StateManager {
	return &StateManager{}
}

func newPersistentStateManager(sessionID string, service *Service) *StateManager {
	return &StateManager{
		sessionID: sessionID,
		service:   service,
	}
}

// IsActive indica si el modo plan está activo.
func (sm *StateManager) IsActive() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.active
}

// GetPlan devuelve el plan actual.
func (sm *StateManager) GetPlan() *Plan {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.plan
}

// IsApproved indica si el plan fue aprobado.
func (sm *StateManager) IsApproved() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.approved
}

// Enter activa el plan mode y conserva un plan existente si ya lo había.
func (sm *StateManager) Enter() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.active {
		return fmt.Errorf("plan mode is already active")
	}

	sm.active = true
	sm.approved = false

	if err := sm.persistLocked(); err != nil {
		return err
	}

	slog.Info("Plan mode activated")
	return nil
}

// SetPlan guarda el plan generado.
func (sm *StateManager) SetPlan(plan *Plan) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.active {
		return fmt.Errorf("plan mode is not active")
	}
	if plan == nil {
		return fmt.Errorf("plan cannot be nil")
	}

	plan.Normalize()
	for i := range plan.Steps {
		if !plan.Steps[i].Status.IsValid() {
			return fmt.Errorf("step %d has invalid status %q", plan.Steps[i].Number, plan.Steps[i].Status)
		}
	}

	now := time.Now()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now
	sm.plan = plan

	if err := sm.persistLocked(); err != nil {
		return err
	}

	slog.Info("Plan set", "steps", len(plan.Steps))
	return nil
}

// Approve marca el plan como aprobado.
func (sm *StateManager) Approve() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.active {
		return fmt.Errorf("plan mode is not active")
	}
	if sm.plan == nil {
		return fmt.Errorf("no plan to approve")
	}
	if sm.approved {
		return fmt.Errorf("plan is already approved")
	}

	sm.approved = true
	sm.plan.UpdatedAt = time.Now()
	if err := sm.persistLocked(); err != nil {
		return err
	}
	slog.Info("Plan approved", "steps", len(sm.plan.Steps))
	return nil
}

// Reject rechaza el plan y sale del modo plan.
func (sm *StateManager) Reject(reason string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.active {
		return fmt.Errorf("plan mode is not active")
	}

	slog.Info("Plan rejected", "reason", reason)
	sm.active = false
	sm.approved = false
	sm.plan = nil
	if err := sm.deleteLocked(); err != nil {
		return err
	}
	return nil
}

// Exit sale del modo plan y devuelve el plan aprobado, si existe.
func (sm *StateManager) Exit() (*Plan, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.active {
		return nil, fmt.Errorf("plan mode is not active")
	}

	var approvedPlan *Plan
	if sm.approved && sm.plan != nil {
		approvedPlan = sm.plan
		approvedPlan.UpdatedAt = time.Now()
		if next := approvedPlan.NextPendingStep(); next != nil {
			next.Status = StepInProgress
		}
	}

	sm.active = false
	sm.approved = false
	if err := sm.persistLocked(); err != nil {
		return nil, err
	}

	slog.Info("Plan mode exited", "has_plan", approvedPlan != nil)
	return approvedPlan, nil
}

func (sm *StateManager) persistLocked() error {
	if sm.service == nil || sm.sessionID == "" {
		return nil
	}
	return sm.service.save(sm.sessionID, sm.active, sm.approved, sm.plan)
}

func (sm *StateManager) deleteLocked() error {
	if sm.service == nil || sm.sessionID == "" {
		return nil
	}
	return sm.service.delete(sm.sessionID)
}
