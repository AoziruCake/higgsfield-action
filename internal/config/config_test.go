package config

import (
	"testing"
	"time"
)

func TestFromEnv_valid(t *testing.T) {
	t.Setenv("INPUT_API_KEY", "key-id:secret")
	t.Setenv("INPUT_PROMPT", "A cat")
	t.Setenv("INPUT_OUTPUT", "out.png")
	t.Setenv("INPUT_MODEL", "")
	t.Setenv("INPUT_ASPECT_RATIO", "")
	t.Setenv("INPUT_RESOLUTION", "")
	t.Setenv("INPUT_TIMEOUT", "5m")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}

	if cfg.Model != DefaultModel {
		t.Fatalf("Model = %q, want %q", cfg.Model, DefaultModel)
	}
	if cfg.AspectRatio != "4:3" {
		t.Fatalf("AspectRatio = %q, want 4:3", cfg.AspectRatio)
	}
	if cfg.Resolution != "720p" {
		t.Fatalf("Resolution = %q, want 720p", cfg.Resolution)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Fatalf("Timeout = %v, want 5m", cfg.Timeout)
	}
}

func TestFromEnv_invalidAPIKey(t *testing.T) {
	t.Setenv("INPUT_API_KEY", "no-colon")
	t.Setenv("INPUT_PROMPT", "x")
	t.Setenv("INPUT_OUTPUT", "out.png")

	_, err := FromEnv()
	if err == nil {
		t.Fatal("expected error for invalid api-key")
	}
}

func TestFromEnv_invalidTimeout(t *testing.T) {
	t.Setenv("INPUT_API_KEY", "a:b")
	t.Setenv("INPUT_PROMPT", "x")
	t.Setenv("INPUT_OUTPUT", "out.png")
	t.Setenv("INPUT_TIMEOUT", "not-a-duration")

	_, err := FromEnv()
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}
