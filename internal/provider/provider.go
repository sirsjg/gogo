package provider

import (
	"context"
	"errors"
	"io"
	"os"

	"gogo/internal/config"
	"gogo/internal/plugin"
)

type Client struct {
	cfg    config.Config
	stderr io.Writer
	tools  *plugin.Registry
}

func NewClient(cfg config.Config, stderr io.Writer, tools *plugin.Registry) *Client {
	return &Client{cfg: cfg, stderr: stderr, tools: tools}
}

func (c *Client) Stream(ctx context.Context, prompt string, out io.Writer) error {
	switch c.cfg.Provider {
	case "openai":
		return streamOpenAI(ctx, c.cfg, prompt, out, c.stderr, c.tools)
	case "anthropic":
		return streamAnthropic(ctx, c.cfg, prompt, out, c.stderr, c.tools)
	case "gemini":
		return streamGemini(ctx, c.cfg, prompt, out, c.stderr, c.tools)
	default:
		return errors.New("unknown provider: " + c.cfg.Provider)
	}
}

func apiKey(env string) (string, error) {
	v := os.Getenv(env)
	if v == "" {
		return "", errors.New("missing " + env)
	}
	return v, nil
}
