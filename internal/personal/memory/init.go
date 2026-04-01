package memory

import (
	"log/slog"
	"sync"
	"time"
)

var (
	instance     *MemoryManager
	instanceOnce sync.Once
	scanner      *Scanner
	detector     *PatternDetector
)

// Init inicializa el sistema de memorias. Se llama una vez al iniciar la app.
// Retorna el MemoryManager inicializado.
func Init(projectDir string) (*MemoryManager, error) {
	var initErr error
	instanceOnce.Do(func() {
		instance, initErr = NewMemoryManager(projectDir)
		if initErr != nil {
			return
		}
		scanner = NewScanner(instance)
		detector = NewPatternDetector(3) // 3 ocurrencias para sugerir

		slog.Info("Memory system initialized",
			"project_dir", projectDir,
			"project_mem", instance.projectMemDir,
			"global_mem", instance.globalMemDir,
		)

		// Limpiar memorias muy antiguas al iniciar (background)
		go func() {
			ager := NewAger(instance, 180*24*time.Hour, false) // 6 meses
			cleaned, err := ager.Clean()
			if err != nil {
				slog.Warn("Failed to clean stale memories", "error", err)
				return
			}
			if cleaned > 0 {
				slog.Info("Cleaned stale memories on startup", "count", cleaned)
			}
		}()
	})
	return instance, initErr
}

// GetManager retorna el MemoryManager singleton.
func GetManager() *MemoryManager {
	return instance
}

// GetScanner retorna el Scanner singleton.
func GetScanner() *Scanner {
	return scanner
}

// GetDetector retorna el PatternDetector singleton.
func GetDetector() *PatternDetector {
	return detector
}
