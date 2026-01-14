package plugin

import (
	"encoding/json"

	"gogo/internal/tool"
)

// FSToolName is the name of the built-in filesystem tool.
const FSToolName = "fs"

// BuiltinFS creates a plugin wrapper for the built-in filesystem tool.
func BuiltinFS() *Tool {
	return &Tool{
		Name:        FSToolName,
		Description: "Filesystem operations (read/write/append/delete/mkdir/rmdir/list/stat/move/copy)",
		Type:        "builtin",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"op":   map[string]string{"type": "string", "description": "Operation: read, write, append, delete, mkdir, rmdir, list, stat, move, copy"},
				"path": map[string]string{"type": "string", "description": "File or directory path"},
				"data": map[string]string{"type": "string", "description": "Data to write (for write/append)"},
				"dest": map[string]string{"type": "string", "description": "Destination path (for move/copy)"},
			},
			"required": []string{"op", "path"},
		},
	}
}

// ExecuteFS runs the built-in filesystem tool.
func ExecuteFS(input []byte) Result {
	var req tool.FSRequest
	if err := json.Unmarshal(input, &req); err != nil {
		return Result{OK: false, Error: err.Error()}
	}
	fsResult := tool.FS(req)
	return Result{
		OK:    fsResult.OK,
		Data:  fsResult.Data,
		Error: fsResult.Error,
	}
}

// LoadWithBuiltins loads user plugins and adds built-in tools.
func LoadWithBuiltins() (*Registry, error) {
	reg, err := LoadDefault()
	if err != nil {
		return nil, err
	}

	// Add built-in fs tool (can be overridden by user plugins)
	fs := BuiltinFS()
	fs.Type = "builtin" // Mark as builtin for special handling
	reg.tools[FSToolName] = fs

	return reg, nil
}

// ExecuteBuiltin handles execution of built-in tools.
func ExecuteBuiltin(name string, input []byte) (Result, bool) {
	switch name {
	case FSToolName:
		return ExecuteFS(input), true
	default:
		return Result{}, false
	}
}
