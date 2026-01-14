package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"gogo/internal/config"
	"gogo/internal/stream"
	"gogo/internal/tool"
)

const anthropicURL = "https://api.anthropic.com/v1/messages"
const anthropicVersion = "2023-06-01"

type anthropicRequest struct {
	Model       string                   `json:"model"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
	Stream      bool                     `json:"stream"`
	Messages    []map[string]interface{} `json:"messages"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
	System      string                   `json:"system,omitempty"`
}

type anthropicEvent struct {
	Type         string          `json:"type"`
	Delta        json.RawMessage `json:"delta"`
	ContentBlock json.RawMessage `json:"content_block"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type anthropicTextDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicInputDelta struct {
	Type    string `json:"type"`
	Partial string `json:"partial_json"`
}

type toolUse struct {
	ID    string
	Name  string
	Input string
}

func streamAnthropic(ctx context.Context, cfg config.Config, prompt string, out io.Writer, stderr io.Writer) error {
	key, err := apiKey("ANTHROPIC_API_KEY")
	if err != nil {
		return err
	}

	messages := []map[string]interface{}{
		{
			"role": "user",
			"content": []map[string]string{
				{"type": "text", "text": prompt},
			},
		},
	}

	return anthropicStreamLoop(ctx, cfg, key, messages, out, stderr)
}

func anthropicStreamLoop(ctx context.Context, cfg config.Config, key string, messages []map[string]interface{}, out io.Writer, stderr io.Writer) error {
	toolUses, err := anthropicStreamOnce(ctx, cfg, key, messages, out)
	if err != nil {
		return err
	}
	if len(toolUses) == 0 {
		return nil
	}

	toolResults := make([]map[string]interface{}, 0, len(toolUses))
	for _, use := range toolUses {
		if use.Name != "fs" {
			continue
		}
		var req tool.FSRequest
		if err := json.Unmarshal([]byte(use.Input), &req); err != nil {
			continue
		}
		res := tool.FS(req)
		logTool(stderr, "anthropic", req, res)
		toolResults = append(toolResults, map[string]interface{}{
			"type":        "tool_result",
			"tool_use_id": use.ID,
			"content":     []map[string]string{{"type": "text", "text": toJSON(res)}},
		})
	}

	if len(toolResults) == 0 {
		return nil
	}

	next := append([]map[string]interface{}{}, messages...)
	next = append(next, map[string]interface{}{
		"role":    "user",
		"content": toolResults,
	})
	_, err = anthropicStreamOnce(ctx, cfg, key, next, out)
	return err
}

func anthropicStreamOnce(ctx context.Context, cfg config.Config, key string, messages []map[string]interface{}, out io.Writer) ([]toolUse, error) {
	reqBody := anthropicRequest{
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Stream:      true,
		Messages:    messages,
	}
	reqBody.System = fsInstruction()

	reqBody.Tools = []map[string]interface{}{
		{
			"name":        "fs",
			"description": "Filesystem operations (read/write/append/delete/mkdir/rmdir/list/stat/move/copy)",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"op":   map[string]string{"type": "string"},
					"path": map[string]string{"type": "string"},
					"data": map[string]string{"type": "string"},
					"dest": map[string]string{"type": "string"},
				},
				"required": []string{"op", "path"},
			},
		},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicURL, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	httpClient := &http.Client{Timeout: 0}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(body))
	}

	writer := bufio.NewWriter(out)
	toolUses := map[string]*toolUse{}
	var activeToolID string

	err = stream.ReadEvents(resp.Body, func(data string) error {
		var event anthropicEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return err
		}

		switch event.Type {
		case "content_block_start":
			var block anthropicContentBlock
			if err := json.Unmarshal(event.ContentBlock, &block); err != nil {
				return err
			}
			if block.Type == "tool_use" {
				toolUses[block.ID] = &toolUse{ID: block.ID, Name: block.Name}
				activeToolID = block.ID
			}
		case "content_block_delta":
			var delta struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(event.Delta, &delta); err != nil {
				return err
			}
			if delta.Type == "text_delta" {
				var text anthropicTextDelta
				if err := json.Unmarshal(event.Delta, &text); err != nil {
					return err
				}
				if text.Text != "" {
					if _, err := writer.WriteString(text.Text); err != nil {
						return err
					}
					if err := writer.Flush(); err != nil {
						return err
					}
				}
			}
			if delta.Type == "input_json_delta" {
				var input anthropicInputDelta
				if err := json.Unmarshal(event.Delta, &input); err != nil {
					return err
				}
				if activeToolID != "" {
					if use := toolUses[activeToolID]; use != nil {
						use.Input += input.Partial
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	uses := make([]toolUse, 0, len(toolUses))
	for _, use := range toolUses {
		uses = append(uses, *use)
	}
	return uses, nil
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
