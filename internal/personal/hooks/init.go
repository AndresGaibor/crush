package hooks

import (
	"log/slog"
	"sync"
)

var (
	instance     *Manager
	instanceLock sync.Mutex
	initialized  bool
)

// Init inicializa el sistema de hooks.
// Se llama una vez al iniciar la app, después de cargar la configuración.
func Init(cwd string, config HookConfigMap) *Manager {
	instanceLock.Lock()
	defer instanceLock.Unlock()

	if initialized {
		return instance
	}

	instance = NewManager(cwd, config)

	counts := instance.HookCount()
	total := 0
	for _, c := range counts {
		total += c
	}
	slog.Info("Hook system initialized",
		"cwd", cwd,
		"total_hooks", total,
		"pre_tool_use", counts[PreToolUse],
		"post_tool_use", counts[PostToolUse],
		"session_start", counts[SessionStart],
		"stop", counts[Stop],
	)

	initialized = true
	return instance
}

// GetManager retorna el Manager singleton.
func GetManager() *Manager {
	instanceLock.Lock()
	defer instanceLock.Unlock()
	return instance
}

// IsInitialized retorna true si el sistema de hooks fue inicializado.
func IsInitialized() bool {
	instanceLock.Lock()
	defer instanceLock.Unlock()
	return initialized
}

// Reset reinicia el singleton (para tests).
func Reset() {
	instanceLock.Lock()
	defer instanceLock.Unlock()
	initialized = false
	instance = nil
}

