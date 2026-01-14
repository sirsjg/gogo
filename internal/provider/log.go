package provider

import (
	"fmt"
	"io"

	"gogo/internal/plugin"
)

// logToolResult logs tool execution for any tool type.
func logToolResult(w io.Writer, provider string, toolName string, input string, res plugin.Result) {
	if w == nil {
		return
	}
	errText := res.Error
	if errText == "" {
		errText = "-"
	}
	// Truncate input if too long for logging
	if len(input) > 100 {
		input = input[:97] + "..."
	}
	fmt.Fprintf(w, "tool %s provider=%s ok=%t err=%s input=%s\n", toolName, provider, res.OK, errText, input)
}
