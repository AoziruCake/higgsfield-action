// Package workspace resolves Action output paths inside the job workspace.
package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// ResolveOutputPath maps an Action output path to an absolute filesystem path.
//
// Relative paths are resolved from GITHUB_WORKSPACE (or the process cwd locally).
// The result must be a file path inside the workspace. ".." and any symlink
// along the path are rejected, so a link cannot redirect the write outside.
func ResolveOutputPath(output string) (string, error) {
	return resolveInsideWorkspace(output, "output is required", "output path is invalid", "output path")
}

// maxPromptFileBytes is the prompt-file size limit (16 KiB).
const maxPromptFileBytes = 16 * 1024

// ReadPromptFile reads a UTF-8 prompt from a regular file inside the workspace.
// The path uses the same confinement as an output file: ".." and any symlink
// are rejected. The returned text is space-trimmed and non-empty.
func ReadPromptFile(path string) (string, error) {
	abs, err := resolveInsideWorkspace(path, "prompt-file is required", "prompt-file path is invalid", "prompt-file")
	if err != nil {
		return "", err
	}

	info, err := os.Lstat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("prompt-file not found")
		}
		return "", fmt.Errorf("prompt-file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("prompt-file must be a regular file")
	}
	if info.Size() > maxPromptFileBytes {
		return "", fmt.Errorf("prompt-file exceeds 16 KiB")
	}

	file, err := os.Open(abs)
	if err != nil {
		return "", fmt.Errorf("prompt-file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxPromptFileBytes+1))
	if err != nil {
		return "", fmt.Errorf("prompt-file: %w", err)
	}
	if int64(len(data)) > maxPromptFileBytes {
		return "", fmt.Errorf("prompt-file exceeds 16 KiB")
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("prompt-file must be UTF-8 text")
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", fmt.Errorf("prompt-file is empty")
	}
	return text, nil
}

// resolveInsideWorkspace maps path to an absolute file path inside the workspace.
// emptyMsg and invalidMsg are the blank and NUL errors. label names the path in
// confinement errors, so a prompt file does not reuse output wording.
func resolveInsideWorkspace(path, emptyMsg, invalidMsg, label string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s", emptyMsg)
	}
	if strings.ContainsRune(path, 0) {
		return "", fmt.Errorf("%s", invalidMsg)
	}

	root, err := workspaceRoot()
	if err != nil {
		return "", err
	}

	var abs string
	if filepath.IsAbs(path) {
		abs = filepath.Clean(path)
	} else {
		abs = filepath.Clean(filepath.Join(root, path))
	}
	if err := walkComponents(root, abs, false, nil, false, label); err != nil {
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
	return walkComponents(root, filepath.Clean(dir), true, func(current string) error {
		return os.Mkdir(current, 0o755)
	}, true, "output path")
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
	return walkComponents(root, abs, allowRoot, nil, false, "output path")
}

// walkComponents applies one path walk to both checks and directory creation.
// onMissing is nil when a missing component ends the walk. requireDir rejects
// an existing file where a directory is required.
func walkComponents(root, abs string, allowRoot bool, onMissing func(current string) error, requireDir bool, label string) error {
	if err := lexicalInside(root, abs, allowRoot, label); err != nil {
		return err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if rel == "." {
		return nil
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
				if onMissing == nil {
					return nil
				}
				if err := onMissing(current); err != nil {
					return err
				}
				continue
			}
			return fmt.Errorf("%s: %w", label, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s must not contain a symlink", label)
		}
		if requireDir && !info.IsDir() {
			return fmt.Errorf("create output directory: not a directory")
		}
		resolved, err := filepath.EvalSymlinks(current)
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		if err := lexicalInside(resolvedRoot, resolved, true, label); err != nil {
			return err
		}
	}
	return nil
}

func lexicalInside(root, abs string, allowRoot bool, label string) error {
	rel, err := filepath.Rel(root, filepath.Clean(abs))
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if rel == "." {
		if allowRoot {
			return nil
		}
		return fmt.Errorf("%s must be a file inside the workspace", label)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%s must be inside the workspace", label)
	}
	return nil
}
