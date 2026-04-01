package tasks

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
)

var (
	initOnce sync.Once
	dbMu     sync.RWMutex
	registry map[string]*Service
	sharedDB *sql.DB
)

// Init prepara el subsistema de tareas.
func Init(db *sql.DB) error {
	var initErr error
	initOnce.Do(func() {
		if db == nil {
			initErr = fmt.Errorf("database connection is nil")
			return
		}

		sharedDB = db
		registry = make(map[string]*Service)

		svc := newService(db)
		if err := svc.ensureSchema(); err != nil {
			initErr = err
			return
		}

		slog.Info("Tasks system initialized")
	})
	return initErr
}

// GetService devuelve el servicio para una sesión.
func GetService(sessionID string) *Service {
	if sessionID == "" || sharedDB == nil || registry == nil {
		return nil
	}

	dbMu.RLock()
	svc, ok := registry[sessionID]
	dbMu.RUnlock()
	if ok {
		return svc
	}

	dbMu.Lock()
	defer dbMu.Unlock()
	if svc, ok := registry[sessionID]; ok {
		return svc
	}

	svc = newService(sharedDB)
	registry[sessionID] = svc
	return svc
}

// Reset limpia el estado global para tests.
func Reset() {
	dbMu.Lock()
	defer dbMu.Unlock()
	initOnce = sync.Once{}
	registry = nil
	sharedDB = nil
}
