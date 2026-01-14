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
	"gogo/internal/plugin"
	"gogo/internal/stream"
)

const openAIURL = "https://api.openai.com/v1/responses"

type openAIRequest struct {
	Model              string           `json:"model"`
	Input              []any            `json:"input"`
	MaxOutputTokens    int              `json:"max_output_tokens,omitempty"`
	Temperature        float64          `json:"temperature,omitempty"`
	Stream             bool             `json:"stream"`
	Tools              []map[string]any `json:"tools,omitempty"`
	ToolChoice         string           `json:"tool_choice,omitempty"`
	PreviousResponseID string           `json:"previous_response_id,omitempty"`
}

type responseEvent struct {
	Type string `json:"type"`
}

type responseCreated struct {
	Response struct {
		ID string `json:"id"`
	} `json:"response"`
}

type outputTextDelta struct {
	Delta string `json:"delta"`
}

type outputItemAdded struct {
	Item responseOutputItem `json:"item"`
}

type responseOutputItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type functionArgsDelta struct {
	ItemID string `json:"item_id"`
	Delta  string `json:"delta"`
}

type toolCall struct {
	ID        string
	CallID    string
	Name      string
	Arguments string
}

func streamOpenAI(ctx context.Context, cfg config.Config, prompt string, out io.Writer, stderr io.Writer, tools *plugin.Registry) error {
	key, err := apiKey("OPENAI_API_KEY")
	if err != nil {
		return err
	}

	input := []any{
		map[string]any{
			"role": "system",
			"content": []map[string]string{
				{"type": "input_text", "text": tools.GenerateInstruction()},
			},
		},
		map[string]any{
			"role": "user",
			"content": []map[string]string{
				{"type": "input_text", "text": prompt},
			},
		},
	}

	return openAIStreamLoop(ctx, cfg, key, input, out, stderr, tools)
}

func openAIStreamLoop(ctx context.Context, cfg config.Config, key string, input []any, out io.Writer, stderr io.Writer, tools *plugin.Registry) error {
	toolCalls, responseID, err := openAIStreamOnce(ctx, cfg, key, input, out, "", tools)
	if err != nil {
		return err
	}

	if len(toolCalls) == 0 {
		return nil
	}

	toolMessages := make([]any, 0, len(toolCalls))
	for _, call := range toolCalls {
		// Check if the tool exists in the registry
		if _, ok := tools.Get(call.Name); !ok {
			continue
		}
		res := tools.ExecuteTool(call.Name, []byte(call.Arguments))
		logToolResult(stderr, "openai", call.Name, call.Arguments, res)
		toolMessages = append(toolMessages, map[string]any{
			"type":    "function_call_output",
			"call_id": call.CallID,
			"output":  res.ToJSON(),
		})
	}

	if len(toolMessages) == 0 {
		return nil
	}

	_, _, err = openAIStreamOnce(ctx, cfg, key, toolMessages, out, responseID, tools)
	return err
}

func openAIStreamOnce(ctx context.Context, cfg config.Config, key string, input []any, out io.Writer, previousID string, tools *plugin.Registry) ([]toolCall, string, error) {
	reqBody := openAIRequest{
		Model:              cfg.Model,
		Input:              input,
		MaxOutputTokens:    cfg.MaxTokens,
		Temperature:        cfg.Temperature,
		Stream:             true,
		PreviousResponseID: previousID,
		Tools:              tools.FormatOpenAITools(),
		ToolChoice:         "auto",
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIURL, bytes.NewReader(b))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 0}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", errors.New(string(body))
	}

	writer := bufio.NewWriter(out)
	toolCalls := make(map[string]*toolCall)
	responseID := ""

	err = stream.ReadEvents(resp.Body, func(data string) error {
		var evt responseEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			return err
		}

		switch evt.Type {
		case "response.created":
			var created responseCreated
			if err := json.Unmarshal([]byte(data), &created); err != nil {
				return err
			}
			responseID = created.Response.ID
		case "response.output_text.delta":
			var delta outputTextDelta
			if err := json.Unmarshal([]byte(data), &delta); err != nil {
				return err
			}
			if delta.Delta != "" {
				if _, err := writer.WriteString(delta.Delta); err != nil {
					return err
				}
				if err := writer.Flush(); err != nil {
					return err
				}
			}
		case "response.output_item.added":
			var added outputItemAdded
			if err := json.Unmarshal([]byte(data), &added); err != nil {
				return err
			}
			if added.Item.Type == "function_call" && added.Item.ID != "" {
				toolCalls[added.Item.ID] = &toolCall{
					ID:        added.Item.ID,
					CallID:    added.Item.CallID,
					Name:      added.Item.Name,
					Arguments: added.Item.Arguments,
				}
			}
		case "response.function_call_arguments.delta":
			var delta functionArgsDelta
			if err := json.Unmarshal([]byte(data), &delta); err != nil {
				return err
			}
			if call := toolCalls[delta.ItemID]; call != nil {
				call.Arguments += delta.Delta
			}
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	calls := make([]toolCall, 0, len(toolCalls))
	for _, call := range toolCalls {
		calls = append(calls, *call)
	}
	return calls, responseID, nil
}
