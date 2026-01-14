package plugin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegistryBasics(t *testing.T) {
	reg := NewRegistry()

	// Test registering a valid HTTP tool
	httpTool := &Tool{
		Name:        "test-http",
		Description: "Test HTTP tool",
		Type:        "http",
		URL:         "https://example.com/api",
	}
	if err := reg.Register(httpTool); err != nil {
		t.Errorf("failed to register http tool: %v", err)
	}

	// Test registering a valid exec tool
	execTool := &Tool{
		Name:        "test-exec",
		Description: "Test exec tool",
		Type:        "exec",
		Command:     "echo",
		Args:        []string{"hello"},
	}
	if err := reg.Register(execTool); err != nil {
		t.Errorf("failed to register exec tool: %v", err)
	}

	// Test getting tools
	if tool, ok := reg.Get("test-http"); !ok || tool.Name != "test-http" {
		t.Error("failed to get test-http tool")
	}
	if tool, ok := reg.Get("test-exec"); !ok || tool.Name != "test-exec" {
		t.Error("failed to get test-exec tool")
	}
	if _, ok := reg.Get("nonexistent"); ok {
		t.Error("should not find nonexistent tool")
	}

	// Test All()
	all := reg.All()
	if len(all) != 2 {
		t.Errorf("expected 2 tools, got %d", len(all))
	}

	// Test Names()
	names := reg.Names()
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}
}

func TestRegistryValidation(t *testing.T) {
	reg := NewRegistry()

	// Empty name
	if err := reg.Register(&Tool{Type: "http", URL: "http://example.com"}); err == nil {
		t.Error("should reject tool without name")
	}

	// Invalid type
	if err := reg.Register(&Tool{Name: "test", Type: "invalid"}); err == nil {
		t.Error("should reject tool with invalid type")
	}

	// HTTP without URL
	if err := reg.Register(&Tool{Name: "test", Type: "http"}); err == nil {
		t.Error("should reject http tool without url")
	}

	// Exec without command
	if err := reg.Register(&Tool{Name: "test", Type: "exec"}); err == nil {
		t.Error("should reject exec tool without command")
	}
}

func TestHTTPToolExecution(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the request is formed correctly
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse the body
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}

		// Return a response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"message": body["message"],
		})
	}))
	defer server.Close()

	tool := &Tool{
		Name:        "test-api",
		Description: "Test API",
		Type:        "http",
		URL:         server.URL,
		Method:      "POST",
		TimeoutMS:   5000,
	}

	input, _ := json.Marshal(map[string]string{"message": "hello"})
	result := tool.Execute(input)

	if !result.OK {
		t.Errorf("expected OK, got error: %s", result.Error)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Errorf("expected map result, got %T", result.Data)
	}
	if data["status"] != "ok" {
		t.Errorf("expected status ok, got %v", data["status"])
	}
}

func TestExecToolExecution(t *testing.T) {
	tool := &Tool{
		Name:        "test-echo",
		Description: "Test echo",
		Type:        "exec",
		Command:     "echo",
		Args:        []string{"{{.message}}"},
		TimeoutMS:   5000,
	}

	input, _ := json.Marshal(map[string]string{"message": "hello world"})
	result := tool.Execute(input)

	if !result.OK {
		t.Errorf("expected OK, got error: %s", result.Error)
	}

	output, ok := result.Data.(string)
	if !ok {
		t.Errorf("expected string result, got %T", result.Data)
	}
	if output != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", output)
	}
}

func TestTemplateSubstitution(t *testing.T) {
	tests := []struct {
		template string
		params   map[string]interface{}
		expected string
	}{
		{
			template: "Hello {{.name}}!",
			params:   map[string]interface{}{"name": "World"},
			expected: "Hello World!",
		},
		{
			template: "{{.a}} and {{.b}}",
			params:   map[string]interface{}{"a": "foo", "b": "bar"},
			expected: "foo and bar",
		},
		{
			template: "No placeholders",
			params:   map[string]interface{}{"unused": "value"},
			expected: "No placeholders",
		},
		{
			template: "{{.num}} items",
			params:   map[string]interface{}{"num": 42},
			expected: "42 items",
		},
	}

	for _, tc := range tests {
		result := substituteTemplate(tc.template, tc.params)
		if result != tc.expected {
			t.Errorf("substituteTemplate(%q, %v) = %q, want %q",
				tc.template, tc.params, result, tc.expected)
		}
	}
}

func TestToolDefsGeneration(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{
		Name:        "tool1",
		Description: "First tool",
		Type:        "http",
		URL:         "http://example.com",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"arg1": map[string]string{"type": "string"},
			},
		},
	})
	reg.Register(&Tool{
		Name:        "tool2",
		Description: "Second tool",
		Type:        "exec",
		Command:     "test",
	})

	defs := reg.GetToolDefs()
	if len(defs) != 2 {
		t.Errorf("expected 2 tool defs, got %d", len(defs))
	}

	// Check that tool definitions are properly formatted
	for _, def := range defs {
		if def.Name == "" {
			t.Error("tool def has empty name")
		}
		if def.Description == "" {
			t.Error("tool def has empty description")
		}
		if def.InputSchema == nil {
			t.Error("tool def has nil input schema")
		}
	}
}

func TestGenerateInstruction(t *testing.T) {
	reg := NewRegistry()

	// Empty registry
	if instr := reg.GenerateInstruction(); instr != "" {
		t.Errorf("expected empty instruction for empty registry, got %q", instr)
	}

	// Add a tool
	reg.Register(&Tool{
		Name:        "test-tool",
		Description: "A test tool for testing",
		Type:        "http",
		URL:         "http://example.com",
	})

	instr := reg.GenerateInstruction()
	if instr == "" {
		t.Error("expected non-empty instruction")
	}
	if !contains(instr, "test-tool") {
		t.Error("instruction should mention tool name")
	}
	if !contains(instr, "A test tool for testing") {
		t.Error("instruction should include tool description")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
