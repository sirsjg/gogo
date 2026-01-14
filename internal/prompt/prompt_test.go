package prompt

import (
	"os"
	"testing"
)

func TestInlinePrompt(t *testing.T) {
	got, err := Read("hello")
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("unexpected prompt: %q", got)
	}
}

func TestStdinPrompt(t *testing.T) {
	orig := os.Stdin
	defer func() { os.Stdin = orig }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	if _, err := w.Write([]byte("from-stdin")); err != nil {
		t.Fatalf("write error: %v", err)
	}
	_ = w.Close()

	os.Stdin = r
	got, err := Read("")
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if got != "from-stdin" {
		t.Fatalf("unexpected prompt: %q", got)
	}
}

func TestEmptyStdin(t *testing.T) {
	orig := os.Stdin
	defer func() { os.Stdin = orig }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	_ = w.Close()

	os.Stdin = r
	_, err = Read("")
	if err == nil {
		t.Fatalf("expected error on empty stdin")
	}
}

func TestNonTTYNoData(t *testing.T) {
	orig := os.Stdin
	defer func() { os.Stdin = orig }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	_, _ = w.Write([]byte{})
	_ = w.Close()
	os.Stdin = r

	_, err = Read("")
	if err == nil {
		t.Fatalf("expected error on empty stdin")
	}
}
