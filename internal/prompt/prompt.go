package prompt

import (
	"errors"
	"io"
	"os"
)

// HasStdin returns true if stdin has piped input available.
func HasStdin() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice == 0
}

func Read(inline string) (string, error) {
	if inline != "" {
		return inline, nil
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	if stat.Mode()&os.ModeCharDevice != 0 {
		return "", nil
	}

	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}

	if len(b) == 0 {
		return "", errors.New("stdin is empty")
	}

	return string(b), nil
}
