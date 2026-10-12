package config

import (
	"strings"
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

func TestFromEnv_dockerHyphenatedInputs(t *testing.T) {
	t.Setenv("INPUT_API_KEY", "")
	t.Setenv("INPUT_ASPECT_RATIO", "")
	t.Setenv("INPUT_API-KEY", "key-id:secret")
	t.Setenv("INPUT_PROMPT", "A cat")
	t.Setenv("INPUT_OUTPUT", "out.png")
	t.Setenv("INPUT_ASPECT-RATIO", "16:9")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if cfg.APIKey != "key-id:secret" {
		t.Fatalf("APIKey = %q", cfg.APIKey)
	}
	if cfg.AspectRatio != "16:9" {
		t.Fatalf("AspectRatio = %q", cfg.AspectRatio)
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

func TestFromEnv_invalidResolution(t *testing.T) {
	t.Setenv("INPUT_API_KEY", "a:b")
	t.Setenv("INPUT_PROMPT", "x")
	t.Setenv("INPUT_OUTPUT", "out.png")
	t.Setenv("INPUT_RESOLUTION", "4k")

	_, err := FromEnv()
	if err == nil {
		t.Fatal("expected error for invalid resolution")
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

func TestFromEnv_videoDefaults(t *testing.T) {
	t.Setenv("INPUT_API_KEY", "key-id:secret")
	t.Setenv("INPUT_PROMPT", "A cat")
	t.Setenv("INPUT_OUTPUT", "out.mp4")
	t.Setenv("INPUT_MODEL", VideoModel)
	t.Setenv("INPUT_ASPECT_RATIO", "")
	t.Setenv("INPUT_RESOLUTION", "")
	t.Setenv("INPUT_DURATION", "")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if !cfg.IsVideo() {
		t.Fatal("expected video config")
	}
	if cfg.AspectRatio != "16:9" {
		t.Fatalf("AspectRatio = %q, want 16:9", cfg.AspectRatio)
	}
	if cfg.Resolution != "720p" {
		t.Fatalf("Resolution = %q, want 720p", cfg.Resolution)
	}
	if cfg.Duration != 5 {
		t.Fatalf("Duration = %d, want 5", cfg.Duration)
	}
}

func TestFromEnv_videoExplicitValues(t *testing.T) {
	t.Setenv("INPUT_API_KEY", "key-id:secret")
	t.Setenv("INPUT_PROMPT", "A cat")
	t.Setenv("INPUT_OUTPUT", "out.mp4")
	t.Setenv("INPUT_MODEL", VideoModel)
	t.Setenv("INPUT_ASPECT_RATIO", "9:16")
	t.Setenv("INPUT_RESOLUTION", "480p")
	t.Setenv("INPUT_DURATION", "12")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if cfg.Model != VideoModel {
		t.Fatalf("Model = %q", cfg.Model)
	}
	if cfg.AspectRatio != "9:16" || cfg.Resolution != "480p" || cfg.Duration != 12 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestFromEnv_videoRejectsDuration(t *testing.T) {
	for _, duration := range []string{"3", "31", "0"} {
		t.Run(duration, func(t *testing.T) {
			t.Setenv("INPUT_API_KEY", "a:b")
			t.Setenv("INPUT_PROMPT", "x")
			t.Setenv("INPUT_OUTPUT", "out.mp4")
			t.Setenv("INPUT_MODEL", VideoModel)
			t.Setenv("INPUT_DURATION", duration)

			_, err := FromEnv()
			if err == nil {
				t.Fatal("expected error for invalid duration")
			}
		})
	}
}

func TestFromEnv_videoRejectsAspectRatio(t *testing.T) {
	t.Setenv("INPUT_API_KEY", "a:b")
	t.Setenv("INPUT_PROMPT", "x")
	t.Setenv("INPUT_OUTPUT", "out.mp4")
	t.Setenv("INPUT_MODEL", VideoModel)
	t.Setenv("INPUT_ASPECT_RATIO", "2:1")

	_, err := FromEnv()
	if err == nil {
		t.Fatal("expected error for invalid aspect-ratio")
	}
}

func TestFromEnv_videoRejectsResolution(t *testing.T) {
	t.Setenv("INPUT_API_KEY", "a:b")
	t.Setenv("INPUT_PROMPT", "x")
	t.Setenv("INPUT_OUTPUT", "out.mp4")
	t.Setenv("INPUT_MODEL", VideoModel)
	t.Setenv("INPUT_RESOLUTION", "4k")

	_, err := FromEnv()
	if err == nil {
		t.Fatal("expected error for invalid resolution")
	}
}

func TestFromEnv_imageRejectsDuration(t *testing.T) {
	for _, duration := range []string{"5", "0"} {
		t.Run(duration, func(t *testing.T) {
			t.Setenv("INPUT_API_KEY", "a:b")
			t.Setenv("INPUT_PROMPT", "x")
			t.Setenv("INPUT_OUTPUT", "out.png")
			t.Setenv("INPUT_DURATION", duration)

			_, err := FromEnv()
			if err == nil {
				t.Fatal("expected error for duration on image model")
			}
		})
	}
}

func TestFromEnv_imageRejects480p(t *testing.T) {
	t.Setenv("INPUT_API_KEY", "a:b")
	t.Setenv("INPUT_PROMPT", "x")
	t.Setenv("INPUT_OUTPUT", "out.png")
	t.Setenv("INPUT_RESOLUTION", "480p")

	_, err := FromEnv()
	if err == nil {
		t.Fatal("expected error for 480p on image model")
	}
}

func TestFromEnv_unsupportedVideoModel(t *testing.T) {
	for _, model := range []string{
		"bytedance/seedance-2.5/image-to-video",
		"alibaba/wan-3.0/text-to-video",
	} {
		t.Run(model, func(t *testing.T) {
			t.Setenv("INPUT_API_KEY", "a:b")
			t.Setenv("INPUT_PROMPT", "x")
			t.Setenv("INPUT_OUTPUT", "out.mp4")
			t.Setenv("INPUT_MODEL", model)
			t.Setenv("INPUT_DURATION", "5")

			_, err := FromEnv()
			if err == nil || !strings.Contains(err.Error(), "unsupported video model") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
