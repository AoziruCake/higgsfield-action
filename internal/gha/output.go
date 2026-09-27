// Package gha writes step outputs to the GITHUB_OUTPUT file.
package gha

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// OutputWriter appends key/value pairs to the GITHUB_OUTPUT file.
type OutputWriter struct {
	path string
}

// NewOutputWriterFromEnv returns a writer when GITHUB_OUTPUT is set.
//
// Local CLI runs omit the variable and become a no-op so tests can run offline.
// On GitHub-hosted runners GITHUB_ACTIONS=true; missing GITHUB_OUTPUT is an error
// because later workflow steps would silently see empty outputs.
func NewOutputWriterFromEnv() (*OutputWriter, error) {
	path := strings.TrimSpace(os.Getenv("GITHUB_OUTPUT"))
	if os.Getenv("GITHUB_ACTIONS") == "true" && path == "" {
		return nil, fmt.Errorf("GITHUB_OUTPUT is required when running in GitHub Actions")
	}
	return &OutputWriter{path: path}, nil
}

// EnsureWritable opens GITHUB_OUTPUT before any paid API work.
func (w *OutputWriter) EnsureWritable() error {
	if w.path == "" {
		return nil
	}
	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("GITHUB_OUTPUT is not writable: %w", err)
	}
	return f.Close()
}

// Set writes one output entry using the GitHub Actions file format.
func (w *OutputWriter) Set(name, value string) error {
	if w.path == "" {
		return nil
	}

	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open GITHUB_OUTPUT: %w", err)
	}
	defer f.Close()

	// Multiline values must use the name<<DELIMITER heredoc form.
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
