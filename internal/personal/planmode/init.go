package planmode

import (
	"database/sql"
	"log/slog"
	"sync"
)

var (
	registryOnce  sync.Once
	registryMu    sync.RWMutex
	registry      map[string]*StateManager
	sharedService *Service
)

// Init prepara el registro de estados por sesión.
func Init(db *sql.DB) error {
	var initErr error
	registryOnce.Do(func() {
		registry = make(map[string]*StateManager)
		if db != nil {
			sharedService = newService(db)
			if err := sharedService.ensureSchema(); err != nil {
				initErr = err
				return
			}
		}
		slog.Info("Plan mode system initialized")
	})
	return initErr
}

// GetStateManager devuelve el estado de una sesión.
func GetStateManager(sessionID string) *StateManager {
	if sessionID == "" || registry == nil {
		return nil
	}

	registryMu.RLock()
	sm, ok := registry[sessionID]
	service := sharedService
	registryMu.RUnlock()
	if ok {
		return sm
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	if sm, ok := registry[sessionID]; ok {
		return sm
	}

	sm = NewStateManager()
	if service != nil {
		sm = newPersistentStateManager(sessionID, service)
		if state, err := service.load(sessionID); err != nil {
			slog.Warn("Failed to load persisted plan mode state", "session_id", sessionID, "error", err)
		} else if state != nil {
			sm.active = state.Active
			sm.approved = state.Approved
			sm.plan = state.Plan
		}
	}
	registry[sessionID] = sm
	return sm
}

// PeekStateManager devuelve el estado cacheado de una sesión sin cargarlo
// desde la base de datos.
func PeekStateManager(sessionID string) *StateManager {
	if sessionID == "" || registry == nil {
		return nil
	}

	registryMu.RLock()
	defer registryMu.RUnlock()

	return registry[sessionID]
}

// Reset limpia el registro; útil para tests.
func Reset() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = nil
	registryOnce = sync.Once{}
	sharedService = nil
}
