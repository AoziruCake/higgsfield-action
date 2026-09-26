package gha

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// OutputWriter appends key/value pairs to GITHUB_OUTPUT.
type OutputWriter struct {
	path string
}

// NewOutputWriterFromEnv returns a writer when GITHUB_OUTPUT is set.
// Local runs without the variable become a no-op writer.
// GitHub Actions runs require GITHUB_OUTPUT so step outputs are actually published.
func NewOutputWriterFromEnv() (*OutputWriter, error) {
	path := strings.TrimSpace(os.Getenv("GITHUB_OUTPUT"))
	if os.Getenv("GITHUB_ACTIONS") == "true" && path == "" {
		return nil, fmt.Errorf("GITHUB_OUTPUT is required when running in GitHub Actions")
	}
	return &OutputWriter{path: path}, nil
}

// Set writes one output entry using the GitHub Actions format.
func (w *OutputWriter) Set(name, value string) error {
	if w.path == "" {
		return nil
	}

	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open GITHUB_OUTPUT: %w", err)
	}
	defer f.Close()

	if strings.ContainsAny(value, "\n\r") {
		delimiter := randomDelimiter()
		if _, err := fmt.Fprintf(f, "%s<<%s\n%s\n%s\n", name, delimiter, value, delimiter); err != nil {
			return fmt.Errorf("write output %q: %w", name, err)
		}
		return nil
	}

	if _, err := fmt.Fprintf(f, "%s=%s\n", name, value); err != nil {
		return fmt.Errorf("write output %q: %w", name, err)
	}
	return nil
}

func randomDelimiter() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "EOF"
	}
	return "ghadelimiter_" + hex.EncodeToString(b[:])
}
