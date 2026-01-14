package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Flags struct {
	Prompt      string
	Provider    string
	Model       string
	MaxTokens   int
	Temperature float64
	ConfigPath  string
	Timeout     time.Duration
	Version     bool
	Update      bool
	Debug       bool
}

type Config struct {
	Provider    string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
	Debug       bool
}

type fileConfig struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	TimeoutMS   int     `json:"timeout_ms"`
}

func Load(flags Flags) (Config, error) {
	cfg := Config{}

	fcfg, _ := readFileConfig(flags.ConfigPath)
	applyFile(&cfg, fcfg)
	applyEnv(&cfg)
	applyFlags(&cfg, flags)
	applyDefaults(&cfg)

	if cfg.Provider == "" {
		return cfg, errors.New("provider is required")
	}
	if cfg.Model == "" {
		return cfg, errors.New("model is required")
	}

	return cfg, nil
}

func readFileConfig(path string) (fileConfig, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fileConfig{}, err
		}
		path = filepath.Join(home, ".config", "gogo", "config.json")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileConfig{}, nil
		}
		return fileConfig{}, err
	}

	var cfg fileConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return fileConfig{}, err
	}
	return cfg, nil
}

func applyFile(cfg *Config, f fileConfig) {
	if f.Provider != "" {
		cfg.Provider = f.Provider
	}
	if f.Model != "" {
		cfg.Model = f.Model
	}
	if f.MaxTokens > 0 {
		cfg.MaxTokens = f.MaxTokens
	}
	if f.Temperature != 0 {
		cfg.Temperature = f.Temperature
	}
	if f.TimeoutMS > 0 {
		cfg.Timeout = time.Duration(f.TimeoutMS) * time.Millisecond
	}
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("GOGO_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("GOGO_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("GOGO_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxTokens = n
		}
	}
	if v := os.Getenv("GOGO_TEMPERATURE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Temperature = f
		}
	}
	if v := os.Getenv("GOGO_TIMEOUT_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Timeout = time.Duration(n) * time.Millisecond
		}
	}
}

func applyFlags(cfg *Config, f Flags) {
	if f.Provider != "" {
		cfg.Provider = f.Provider
	}
	if f.Model != "" {
		cfg.Model = f.Model
	}
	if f.MaxTokens > 0 {
		cfg.MaxTokens = f.MaxTokens
	}
	if f.Temperature != 0 {
		cfg.Temperature = f.Temperature
	}
	if f.Timeout > 0 {
		cfg.Timeout = f.Timeout
	}
	cfg.Debug = f.Debug
}

func applyDefaults(cfg *Config) {
	if cfg.Provider == "openai" && cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	if cfg.Provider == "anthropic" && cfg.Model == "" {
		cfg.Model = "claude-3-5-haiku-latest"
	}
	if cfg.Provider == "gemini" && cfg.Model == "" {
		cfg.Model = "gemini-1.5-flash"
	}
}
