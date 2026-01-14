package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PluginsConfig is the structure of the plugins.json config file.
type PluginsConfig struct {
	Tools []Tool `json:"tools"`
}

// LoadFromFile loads plugins from a JSON config file.
func LoadFromFile(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewRegistry(), nil
		}
		return nil, err
	}

	var cfg PluginsConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	reg := NewRegistry()
	for i := range cfg.Tools {
		if err := reg.Register(&cfg.Tools[i]); err != nil {
			// Skip invalid tools but continue loading others
			continue
		}
	}

	return reg, nil
}

// LoadDefault loads plugins from the default config location (~/.config/gogo/plugins.json).
func LoadDefault() (*Registry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return NewRegistry(), nil
	}
	path := filepath.Join(home, ".config", "gogo", "plugins.json")
	return LoadFromFile(path)
}

// DefaultPath returns the default plugins config path.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "gogo", "plugins.json")
}
