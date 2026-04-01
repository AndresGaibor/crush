package subagents

import (
	"log/slog"
	"sync"
)

var (
	instance     *Registry
	instanceOnce sync.Once
)

// Init inicializa el registro global de subagentes.
func Init(paths []string) error {
	var initErr error
	instanceOnce.Do(func() {
		instance = NewRegistry(paths)
		initErr = instance.Reload(nil)
		if initErr != nil {
			return
		}
		slog.Info("Subagents subsystem initialized", "count", len(instance.All()))
	})
	return initErr
}

// Reload vuelve a cargar los subagentes del registro global.
func Reload(paths []string) error {
	if instance == nil {
		return ErrNotInitialized
	}
	return instance.Reload(paths)
}

// Get devuelve un subagente por nombre desde el registro global.
func Get(name string) (*Subagent, bool) {
	if instance == nil {
		return nil, false
	}
	return instance.Get(name)
}

// Find intenta resolver un subagente por nombre o por coincidencia.
func Find(query string) (*Subagent, bool) {
	if instance == nil {
		return nil, false
	}
	return instance.Find(query)
}

// List devuelve todos los subagentes cargados.
func List() []*Subagent {
	if instance == nil {
		return nil
	}
	return instance.All()
}

// IsInitialized informa si el registro global está listo.
func IsInitialized() bool {
	return instance != nil
}

// Reset reinicia el singleton para tests.
func Reset() {
	instance = nil
	instanceOnce = sync.Once{}
}
