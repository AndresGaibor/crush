package cron

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

// LoadConfig carga la configuración de cron desde un archivo JSON.
func LoadConfig(path string) CronConfig {
	cfg := DefaultCronConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	var wrapper struct {
		Cron *CronConfig `json:"cron"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		slog.Warn("Failed to parse cron config", "path", path, "error", err)
		return cfg
	}
	if wrapper.Cron == nil {
		return cfg
	}

	if wrapper.Cron.MaxJobs > 0 {
		cfg.MaxJobs = wrapper.Cron.MaxJobs
	}
	if wrapper.Cron.MaxConcurrent > 0 {
		cfg.MaxConcurrent = wrapper.Cron.MaxConcurrent
	}
	if wrapper.Cron.DefaultTimeoutSec > 0 {
		cfg.DefaultTimeoutSec = wrapper.Cron.DefaultTimeoutSec
	}

	return cfg
}

// LoadConfigFromDirs carga la configuración desde múltiples directorios.
func LoadConfigFromDirs(dirs []string) CronConfig {
	cfg := DefaultCronConfig()
	for _, dir := range dirs {
		loaded := LoadConfig(filepath.Join(dir, "cron.json"))
		if loaded.MaxJobs != DefaultCronConfig().MaxJobs {
			cfg.MaxJobs = loaded.MaxJobs
		}
		if loaded.MaxConcurrent != DefaultCronConfig().MaxConcurrent {
			cfg.MaxConcurrent = loaded.MaxConcurrent
		}
		if loaded.DefaultTimeoutSec != DefaultCronConfig().DefaultTimeoutSec {
			cfg.DefaultTimeoutSec = loaded.DefaultTimeoutSec
		}
	}
	return cfg
}
