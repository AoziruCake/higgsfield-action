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
// The result must be a file path inside the workspace. ".." and any symlink
// along the path are rejected, so a link cannot redirect the write outside.
func ResolveOutputPath(output string) (string, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return "", fmt.Errorf("output is required")
	}
	if strings.ContainsRune(output, 0) {
		return "", fmt.Errorf("output path is invalid")
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

	if err := confine(root, abs, false); err != nil {
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
	if err := confine(root, abs, false); err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", fmt.Errorf("output path: %w", err)
	}
	return filepath.ToSlash(rel), nil
}

// MkdirWithin creates dir and its parents inside the workspace.
// It refuses to follow a symlink, so a link cannot create directories outside.
func MkdirWithin(dir string) error {
	root, err := workspaceRoot()
	if err != nil {
		return err
	}
	dir = filepath.Clean(dir)
	if err := confine(root, dir, true); err != nil {
		return err
	}

	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return fmt.Errorf("output path: %w", err)
	}
	if rel == "." {
		return nil
	}

	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				if err := os.Mkdir(current, 0o755); err != nil {
					return err
				}
				continue
			}
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path must not contain a symlink")
		}
		if !info.IsDir() {
			return fmt.Errorf("create output directory: not a directory")
		}
	}
	return nil
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

// ProbeWritable reports whether dir accepts a new file.
// It creates a temporary file and removes it, so a failed generation does not
// leave a zero-byte output behind.
func ProbeWritable(dir string) error {
	file, err := os.CreateTemp(dir, ".higgsfield-action-write-*")
	if err != nil {
		return err
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Remove(name)
}

// confine checks that abs is lexically inside root and that no existing
// component is a symlink. allowRoot permits abs to be root itself, which
// directory creation needs and file outputs do not.
func confine(root, abs string, allowRoot bool) error {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return fmt.Errorf("output path: %w", err)
	}
	if rel == "." {
		if allowRoot {
			return nil
		}
		return fmt.Errorf("output path must be a file inside the workspace")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("output path must be inside the workspace")
	}

	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
	}
	resolvedRoot = filepath.Clean(resolvedRoot)

	current := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil
			}
			return fmt.Errorf("output path: %w", statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path must not contain a symlink")
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return fmt.Errorf("output path: %w", err)
		}
		if err := lexicalInside(resolvedRoot, resolved, true); err != nil {
			return err
		}
	}
	return nil
}

func lexicalInside(root, abs string, allowRoot bool) error {
	rel, err := filepath.Rel(root, filepath.Clean(abs))
	if err != nil {
		return fmt.Errorf("output path: %w", err)
	}
	if rel == "." {
		if allowRoot {
			return nil
		}
		return fmt.Errorf("output path must be a file inside the workspace")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("output path must be inside the workspace")
	}
	return nil
}
