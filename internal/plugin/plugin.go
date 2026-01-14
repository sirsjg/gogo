// Package plugin provides a simple, user-configurable tool system for gogo.
// Users can define custom tools in their config file that make HTTP/API calls
// or execute commands, extending gogo's functionality without code changes.
package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Tool represents a user-configurable tool that can be called by the LLM.
type Tool struct {
	// Name is the unique identifier for this tool
	Name string `json:"name"`

	// Description explains what the tool does (shown to LLM)
	Description string `json:"description"`

	// Type is either "http" for API calls or "exec" for command execution
	Type string `json:"type"`

	// URL is the endpoint for HTTP tools (supports {{.field}} placeholders)
	URL string `json:"url,omitempty"`

	// Method is the HTTP method (GET, POST, PUT, DELETE). Defaults to POST.
	Method string `json:"method,omitempty"`

	// Headers are HTTP headers to include (supports env var substitution with $VAR)
	Headers map[string]string `json:"headers,omitempty"`

	// Body is the request body template for HTTP tools (supports {{.field}} placeholders)
	Body string `json:"body,omitempty"`

	// Command is the executable for exec tools
	Command string `json:"command,omitempty"`

	// Args are command arguments (supports {{.field}} placeholders)
	Args []string `json:"args,omitempty"`

	// InputSchema defines what parameters the tool accepts (JSON Schema format)
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`

	// Timeout in milliseconds (default: 30000)
	TimeoutMS int `json:"timeout_ms,omitempty"`
}

// Result is the standardized response from tool execution.
type Result struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// Registry holds all registered tools.
type Registry struct {
	tools map[string]*Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t *Tool) error {
	if t.Name == "" {
		return errors.New("tool name is required")
	}
	if t.Type != "http" && t.Type != "exec" && t.Type != "builtin" {
		return fmt.Errorf("invalid tool type %q: must be 'http', 'exec', or 'builtin'", t.Type)
	}
	if t.Type == "http" && t.URL == "" {
		return errors.New("url is required for http tools")
	}
	if t.Type == "exec" && t.Command == "" {
		return errors.New("command is required for exec tools")
	}
	r.tools[t.Name] = t
	return nil
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (*Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// All returns all registered tools.
func (r *Registry) All() []*Tool {
	tools := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// Names returns the names of all registered tools.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Execute runs a tool with the given input and returns the result.
func (r *Registry) Execute(name string, input []byte) Result {
	t, ok := r.tools[name]
	if !ok {
		return Result{OK: false, Error: fmt.Sprintf("unknown tool: %s", name)}
	}
	return t.Execute(input)
}

// Execute runs the tool with the given JSON input.
func (t *Tool) Execute(input []byte) Result {
	// Parse input into a map for template substitution
	var params map[string]interface{}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &params); err != nil {
			return Result{OK: false, Error: fmt.Sprintf("invalid input: %v", err)}
		}
	}
	if params == nil {
		params = make(map[string]interface{})
	}

	timeout := time.Duration(t.TimeoutMS) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	switch t.Type {
	case "http":
		return t.executeHTTP(params, timeout)
	case "exec":
		return t.executeExec(params, timeout)
	case "builtin":
		// Builtin tools are handled separately by ExecuteBuiltin
		return Result{OK: false, Error: "builtin tools must be executed via ExecuteBuiltin"}
	default:
		return Result{OK: false, Error: fmt.Sprintf("unknown tool type: %s", t.Type)}
	}
}

func (t *Tool) executeHTTP(params map[string]interface{}, timeout time.Duration) Result {
	// Substitute placeholders in URL
	url := substituteTemplate(t.URL, params)

	// Substitute placeholders in body
	var body io.Reader
	if t.Body != "" {
		bodyStr := substituteTemplate(t.Body, params)
		body = strings.NewReader(bodyStr)
	} else if len(params) > 0 {
		// If no body template but we have params, send as JSON
		b, err := json.Marshal(params)
		if err != nil {
			return Result{OK: false, Error: fmt.Sprintf("failed to marshal params: %v", err)}
		}
		body = bytes.NewReader(b)
	}

	method := t.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return Result{OK: false, Error: fmt.Sprintf("failed to create request: %v", err)}
	}

	// Set headers with env var substitution
	for key, value := range t.Headers {
		req.Header.Set(key, substituteEnvVars(value))
	}

	// Default content type for POST/PUT
	if (method == "POST" || method == "PUT") && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return Result{OK: false, Error: fmt.Sprintf("request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{OK: false, Error: fmt.Sprintf("failed to read response: %v", err)}
	}

	if resp.StatusCode >= 400 {
		return Result{OK: false, Error: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))}
	}

	// Try to parse as JSON, otherwise return as string
	var data interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		data = string(respBody)
	}

	return Result{OK: true, Data: data}
}

func (t *Tool) executeExec(params map[string]interface{}, timeout time.Duration) Result {
	// Substitute placeholders in command and args
	command := substituteTemplate(t.Command, params)
	args := make([]string, len(t.Args))
	for i, arg := range t.Args {
		args[i] = substituteTemplate(arg, params)
	}

	cmd := exec.Command(command, args...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			errMsg := stderr.String()
			if errMsg == "" {
				errMsg = err.Error()
			}
			return Result{OK: false, Error: errMsg}
		}
	case <-time.After(timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return Result{OK: false, Error: "command timed out"}
	}

	output := stdout.String()

	// Try to parse as JSON
	var data interface{}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		data = output
	}

	return Result{OK: true, Data: data}
}

// substituteTemplate replaces {{.field}} placeholders with values from params.
func substituteTemplate(template string, params map[string]interface{}) string {
	result := template
	for key, value := range params {
		placeholder := "{{." + key + "}}"
		var strValue string
		switch v := value.(type) {
		case string:
			strValue = v
		default:
			b, _ := json.Marshal(v)
			strValue = string(b)
		}
		result = strings.ReplaceAll(result, placeholder, strValue)
	}
	return result
}

// substituteEnvVars replaces $VAR and ${VAR} with environment variable values.
func substituteEnvVars(s string) string {
	return os.ExpandEnv(s)
}
