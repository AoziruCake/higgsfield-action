package gha

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutputWriter_Set(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output")
	t.Setenv("GITHUB_OUTPUT", path)

	w, err := NewOutputWriterFromEnv()
	if err != nil {
		t.Fatalf("NewOutputWriterFromEnv: %v", err)
	}
	if err := w.Set("request-id", "abc-123"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := w.Set("image-url", "https://cdn.example/a.png"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "request-id=abc-123\n") {
		t.Fatalf("content = %q", content)
	}
	if !strings.Contains(content, "image-url=https://cdn.example/a.png\n") {
		t.Fatalf("content = %q", content)
	}
}

func TestOutputWriter_noopWithoutEnv(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "")

	w, err := NewOutputWriterFromEnv()
	if err != nil {
		t.Fatalf("NewOutputWriterFromEnv: %v", err)
	}
	if err := w.Set("request-id", "x"); err != nil {
		t.Fatalf("Set: %v", err)
	}
}
