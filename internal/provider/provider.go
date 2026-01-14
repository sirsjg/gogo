package provider

import (
	"context"
	"errors"
	"io"
	"os"

	"gogo/internal/config"
)

type Client struct {
	cfg    config.Config
	stderr io.Writer
}

func NewClient(cfg config.Config, stderr io.Writer) *Client {
	return &Client{cfg: cfg, stderr: stderr}
}

func (c *Client) Stream(ctx context.Context, prompt string, out io.Writer) error {
	switch c.cfg.Provider {
	case "openai":
		return streamOpenAI(ctx, c.cfg, prompt, out, c.stderr)
	case "anthropic":
		return streamAnthropic(ctx, c.cfg, prompt, out, c.stderr)
	case "gemini":
		return streamGemini(ctx, c.cfg, prompt, out, c.stderr)
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
