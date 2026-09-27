package workspace

import (
	"path/filepath"
	"testing"
)

func TestResolveOutputPath_relative(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITHUB_WORKSPACE", dir)

	got, err := ResolveOutputPath("out/image.png")
	if err != nil {
		t.Fatalf("ResolveOutputPath: %v", err)
	}
	want := filepath.Join(dir, "out", "image.png")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRel_workspaceRelativeSlash(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITHUB_WORKSPACE", dir)

	abs, err := ResolveOutputPath("out/image.png")
	if err != nil {
		t.Fatalf("ResolveOutputPath: %v", err)
	}
	rel, err := Rel(abs)
	if err != nil {
		t.Fatalf("Rel: %v", err)
	}
	if rel != "out/image.png" {
		t.Fatalf("Rel = %q, want out/image.png", rel)
	}
}

func TestResolveOutputPath_rejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITHUB_WORKSPACE", dir)

	_, err := ResolveOutputPath("../escape.png")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}
