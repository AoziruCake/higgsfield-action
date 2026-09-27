// Package workspace resolves Action output paths inside the job workspace.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveOutputPath maps an Action output path to an absolute filesystem path.
//
// Relative paths are resolved from GITHUB_WORKSPACE (or the process cwd locally).
// Paths that escape the workspace with ".." are rejected.
func ResolveOutputPath(output string) (string, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return "", fmt.Errorf("output is required")
	}

	root, err := workspaceRoot()
	if err != nil {
		return "", err
	}

	var abs string
	if filepath.IsAbs(output) {
		abs = filepath.Clean(output)
	} else {
		abs = filepath.Clean(filepath.Join(root, output))
	}

	if err := ensureWithinRoot(root, abs); err != nil {
		return "", err
	}
	return abs, nil
}

// Rel returns a workspace-relative path with forward slashes.
// Later GitHub Actions steps run on the host, where /github/workspace does not exist.
func Rel(abs string) (string, error) {
	root, err := workspaceRoot()
	if err != nil {
		return "", err
	}
	if err := ensureWithinRoot(root, abs); err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", fmt.Errorf("output path: %w", err)
	}
	return filepath.ToSlash(rel), nil
}

func workspaceRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv("GITHUB_WORKSPACE"))
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve workspace: %w", err)
		}
	}
	return filepath.Clean(root), nil
}

func ensureWithinRoot(root, abs string) error {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return fmt.Errorf("output path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("output path must be inside the workspace")
	}
	return nil
}
