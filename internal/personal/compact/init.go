package compact

import (
	"log/slog"
	"sync"
)

var (
	instance     *Manager
	instanceOnce sync.Once
)

// Init initializes the compact system.
func Init(projectDir string) *Manager {
	instanceOnce.Do(func() {
		config := LoadConfig(projectDir)
		instance = NewManager(config)
		slog.Info("Compact system initialized",
			"level", config.Level,
			"threshold_pct", config.ThresholdPct,
			"micro_max_lines", config.MicroMaxLines,
			"buffer_tokens", config.BufferTokens,
		)
	})
	return instance
}

// GetManager returns the initialized manager instance.
func GetManager() *Manager {
	return instance
}

// IsInitialized returns true if the system is initialized.
func IsInitialized() bool {
	return instance != nil
}

// Reset resets the system for testing.
func Reset() {
	instanceOnce = sync.Once{}
	instance = nil
}
