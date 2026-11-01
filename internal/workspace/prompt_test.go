package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPromptFile_trims(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITHUB_WORKSPACE", dir)
	path := filepath.Join(dir, "prompts", "latest.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("  from the file \n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadPromptFile("prompts/latest.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "from the file" {
		t.Fatalf("prompt = %q", got)
	}
}

func TestReadPromptFile_exactLimit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITHUB_WORKSPACE", dir)
	body := strings.Repeat("a", maxPromptFileBytes)
	if err := os.WriteFile(filepath.Join(dir, "prompt.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadPromptFile("prompt.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != body {
		t.Fatalf("len = %d, want %d", len(got), len(body))
	}
}

func TestReadPromptFile_rejects(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITHUB_WORKSPACE", dir)
	outside := filepath.Join(filepath.Dir(dir), "outside.txt")
	if err := os.WriteFile(outside, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "empty.txt"), []byte(" \n\t"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "binary.txt"), []byte{0xff, 0xfe}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), []byte(strings.Repeat("a", maxPromptFileBytes+1)), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []string{
		"../escape.txt",
		".",
		outside,
		"missing.txt",
		"dir",
		"empty.txt",
		"binary.txt",
		"big.txt",
		"bad\x00name",
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			_, err := ReadPromptFile(path)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "prompt-file") {
				t.Fatalf("err = %v, want prompt-file", err)
			}
			if strings.Contains(err.Error(), "output path") {
				t.Fatalf("err = %v, reused output path wording", err)
			}
		})
	}
}

func TestReadPromptFile_rejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITHUB_WORKSPACE", dir)
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(target, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "prompt.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink: %v", err)
	}

	_, err := ReadPromptFile("prompt.txt")
	if err == nil {
		t.Fatal("expected error for symlink")
	}
	if !strings.Contains(err.Error(), "prompt-file") || strings.Contains(err.Error(), "output path") {
		t.Fatalf("err = %v", err)
	}
}
