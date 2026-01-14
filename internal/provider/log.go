package provider

import (
	"fmt"
	"io"

	"gogo/internal/tool"
)

func logTool(w io.Writer, provider string, req tool.FSRequest, res tool.FSResult) {
	if w == nil {
		return
	}
	errText := res.Error
	if errText == "" {
		errText = "-"
	}
	fmt.Fprintf(w, "tool fs provider=%s op=%s path=%s ok=%t err=%s\n", provider, req.Op, req.Path, res.OK, errText)
}
