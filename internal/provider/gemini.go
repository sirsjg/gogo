package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"

	"gogo/internal/config"
	"gogo/internal/plugin"
	"gogo/internal/stream"
)

const geminiBase = "https://generativelanguage.googleapis.com/v1beta/models/"

type geminiRequest struct {
	Contents          []geminiContent        `json:"contents"`
	GenerationConfig  map[string]interface{} `json:"generationConfig,omitempty"`
	Tools             []geminiTool           `json:"tools,omitempty"`
	SystemInstruction *geminiSystem          `json:"systemInstruction,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiSystem struct {
	Parts []geminiPart `json:"parts"`
}

type geminiFunctionDecl struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type geminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type geminiEvent struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func streamGemini(ctx context.Context, cfg config.Config, prompt string, out io.Writer, stderr io.Writer, tools *plugin.Registry) error {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		key = os.Getenv("GOOGLE_API_KEY")
	}
	if key == "" {
		return errors.New("missing GEMINI_API_KEY or GOOGLE_API_KEY")
	}

	contents := []geminiContent{
		{Role: "user", Parts: []geminiPart{{Text: prompt}}},
	}

	return geminiStreamLoop(ctx, cfg, key, contents, out, stderr, tools)
}

func geminiStreamLoop(ctx context.Context, cfg config.Config, key string, contents []geminiContent, out io.Writer, stderr io.Writer, tools *plugin.Registry) error {
	calls, err := geminiStreamOnce(ctx, cfg, key, contents, out, tools)
	if err != nil {
		return err
	}
	if len(calls) == 0 {
		return nil
	}

	responses := make([]geminiPart, 0, len(calls))
	for _, call := range calls {
		// Check if the tool exists in the registry
		if _, ok := tools.Get(call.Name); !ok {
			continue
		}
		reqBytes, _ := json.Marshal(call.Args)
		res := tools.ExecuteTool(call.Name, reqBytes)
		logToolResult(stderr, "gemini", call.Name, string(reqBytes), res)
		responses = append(responses, geminiPart{
			FunctionResponse: &geminiFunctionResponse{
				Name:     call.Name,
				Response: map[string]interface{}{"result": res},
			},
		})
	}
	if len(responses) == 0 {
		return nil
	}

	next := append([]geminiContent{}, contents...)
	next = append(next, geminiContent{Role: "function", Parts: responses})
	_, err = geminiStreamOnce(ctx, cfg, key, next, out, tools)
	return err
}

func geminiStreamOnce(ctx context.Context, cfg config.Config, key string, contents []geminiContent, out io.Writer, tools *plugin.Registry) ([]geminiFunctionCall, error) {
	reqBody := geminiRequest{
		Contents: contents,
	}
	if cfg.MaxTokens > 0 || cfg.Temperature > 0 {
		reqBody.GenerationConfig = map[string]interface{}{}
		if cfg.MaxTokens > 0 {
			reqBody.GenerationConfig["maxOutputTokens"] = cfg.MaxTokens
		}
		if cfg.Temperature > 0 {
			reqBody.GenerationConfig["temperature"] = cfg.Temperature
		}
	}
	// Build function declarations from the tool registry
	funcDecls := make([]geminiFunctionDecl, 0)
	for _, def := range tools.GetToolDefs() {
		funcDecls = append(funcDecls, geminiFunctionDecl{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.InputSchema,
		})
	}
	reqBody.Tools = []geminiTool{
		{
			FunctionDeclarations: funcDecls,
		},
	}
	reqBody.SystemInstruction = &geminiSystem{
		Parts: []geminiPart{{Text: tools.GenerateInstruction()}},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	endpoint := geminiBase + url.PathEscape(cfg.Model) + ":streamGenerateContent"
	u, _ := url.Parse(endpoint)
	q := u.Query()
	q.Set("alt", "sse")
	q.Set("key", key)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

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
	var calls []geminiFunctionCall

	err = stream.ReadEvents(resp.Body, func(data string) error {
		var event geminiEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return err
		}
		for _, cand := range event.Candidates {
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					if _, err := writer.WriteString(part.Text); err != nil {
						return err
					}
					if err := writer.Flush(); err != nil {
						return err
					}
				}
				if part.FunctionCall != nil {
					calls = append(calls, *part.FunctionCall)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return calls, nil
}
