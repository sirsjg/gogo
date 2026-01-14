package plugin

import (
	"encoding/json"
)

// ToolDef is the tool definition format used by LLM providers.
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// GetToolDefs returns tool definitions for all registered tools.
// This is used by providers to build their tool registration payloads.
func (r *Registry) GetToolDefs() []ToolDef {
	defs := make([]ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		schema := t.InputSchema
		if schema == nil {
			// Default schema if none provided
			schema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		defs = append(defs, ToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}
	return defs
}

// ExecuteTool runs a tool by name with JSON input bytes.
// It handles both builtin and user-defined tools.
func (r *Registry) ExecuteTool(name string, input []byte) Result {
	t, ok := r.tools[name]
	if !ok {
		return Result{OK: false, Error: "unknown tool: " + name}
	}

	if t.Type == "builtin" {
		res, handled := ExecuteBuiltin(name, input)
		if handled {
			return res
		}
		return Result{OK: false, Error: "unhandled builtin tool: " + name}
	}

	return t.Execute(input)
}

// FormatAnthropicTools formats tools for Anthropic's API.
func (r *Registry) FormatAnthropicTools() []map[string]interface{} {
	tools := make([]map[string]interface{}, 0, len(r.tools))
	for _, t := range r.tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		tools = append(tools, map[string]interface{}{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": schema,
		})
	}
	return tools
}

// FormatOpenAITools formats tools for OpenAI's API.
func (r *Registry) FormatOpenAITools() []map[string]interface{} {
	tools := make([]map[string]interface{}, 0, len(r.tools))
	for _, t := range r.tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		tools = append(tools, map[string]interface{}{
			"type":        "function",
			"name":        t.Name,
			"description": t.Description,
			"parameters":  schema,
		})
	}
	return tools
}

// FormatGeminiTools formats tools for Gemini's API.
// Returns a slice that can be used directly in geminiFunctionDecl structs.
func (r *Registry) FormatGeminiTools() []map[string]interface{} {
	decls := make([]map[string]interface{}, 0, len(r.tools))
	for _, t := range r.tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		decls = append(decls, map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  schema,
		})
	}
	return decls
}

// ToJSON converts a result to JSON string.
func (r Result) ToJSON() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// GenerateInstruction creates a system instruction for all registered tools.
func (r *Registry) GenerateInstruction() string {
	if len(r.tools) == 0 {
		return ""
	}

	instruction := "You have access to the following tools. Use them when appropriate:\n\n"
	for _, t := range r.tools {
		instruction += "- " + t.Name + ": " + t.Description + "\n"
	}
	instruction += "\nCall tools when needed to complete the user's request. Do not claim to have performed actions without using the appropriate tool."
	return instruction
}
