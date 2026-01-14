package stream

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// ReadEvents reads text/event-stream events and yields data payloads.
// It returns when the stream ends or an error occurs.
func ReadEvents(r io.Reader, onData func(string) error) error {
	scanner := bufio.NewScanner(r)
	var buf bytes.Buffer

	flush := func() error {
		if buf.Len() == 0 {
			return nil
		}
		data := buf.String()
		buf.Reset()
		return onData(data)
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if buf.Len() > 0 {
				buf.WriteString("\n")
			}
			buf.WriteString(payload)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return flush()
}
