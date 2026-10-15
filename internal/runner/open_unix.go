//go:build unix

package runner

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// createOutput creates or truncates path without following a final symlink.
func createOutput(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, fmt.Errorf("output path must not be a symlink")
		}
		return nil, err
	}
	return file, nil
}
