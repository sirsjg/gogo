package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"provider":"openai","model":"file-model","max_tokens":10,"temperature":0.1,"timeout_ms":1000}`), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GOGO_PROVIDER", "anthropic")
	t.Setenv("GOGO_MODEL", "env-model")
	t.Setenv("GOGO_MAX_TOKENS", "20")
	t.Setenv("GOGO_TEMPERATURE", "0.2")
	t.Setenv("GOGO_TIMEOUT_MS", "2000")

	flags := Flags{
		Provider:    "gemini",
		Model:       "flag-model",
		MaxTokens:   30,
		Temperature: 0.3,
		Timeout:     3 * time.Second,
		ConfigPath:  path,
	}

	cfg, err := Load(flags)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Provider != "gemini" {
		t.Fatalf("provider precedence failed: %s", cfg.Provider)
	}
	if cfg.Model != "flag-model" {
		t.Fatalf("model precedence failed: %s", cfg.Model)
	}
	if cfg.MaxTokens != 30 {
		t.Fatalf("max tokens precedence failed: %d", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.3 {
		t.Fatalf("temperature precedence failed: %v", cfg.Temperature)
	}
	if cfg.Timeout != 3*time.Second {
		t.Fatalf("timeout precedence failed: %v", cfg.Timeout)
	}
}

func TestDefaults(t *testing.T) {
	cfg, err := Load(Flags{Provider: "openai"})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Fatalf("default model not set: %s", cfg.Model)
	}
}
