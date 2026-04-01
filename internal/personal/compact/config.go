package compact

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LoadConfig loads the compact configuration from crush.json.
func LoadConfig(projectDir string) *CompactConfig {
	config := DefaultConfig()
	paths := []string{
		filepath.Join(projectDir, ".crush", "crush.json"),
		filepath.Join(projectDir, "crush.json"),
		homeDir(),
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		var wrapper struct {
			Options struct {
				Compact json.RawMessage `json:"compact"`
			} `json:"options"`
		}

		if err := json.Unmarshal(data, &wrapper); err != nil {
			continue
		}

		if wrapper.Options.Compact != nil {
			if err := json.Unmarshal(wrapper.Options.Compact, config); err == nil {
				return config
			}
		}
	}

	return config
}

func homeDir() string {
	d, _ := os.UserHomeDir()
	if d == "" {
		d = os.TempDir()
	}
	configPath := filepath.Join(d, ".config", "crush", "crush.json")
	return configPath
}
