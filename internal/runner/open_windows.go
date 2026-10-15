//go:build windows

package runner

import (
	"fmt"
	"os"
)

// createOutput creates or truncates path. Windows has no O_NOFOLLOW, so a
// symlink at the final component is rejected with Lstat before the open.
func createOutput(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("output path must not be a symlink")
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
}
