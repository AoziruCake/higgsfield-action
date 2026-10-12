// Package config reads GitHub Action inputs from INPUT_* environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultModel is Soul v2 Standard, the image endpoint used when model is unset.
	DefaultModel = "higgsfield-ai/soul/v2/standard"
	// VideoModel is Seedance 2.5 text-to-video, the only supported video endpoint.
	VideoModel = "bytedance/seedance-2.5/text-to-video"
)

// Config holds GitHub Action inputs after parsing and validation.
//
// Docker Actions set INPUT_API-KEY (hyphen kept). Some runners and local
// tests use INPUT_API_KEY. FromEnv accepts both.
type Config struct {
	APIKey      string // KEY_ID:KEY_SECRET
	Prompt      string
	Output      string // workspace-relative path
	Model       string
	AspectRatio string
	Resolution  string
	Duration    int // seconds; video only. Zero means unset for image models.
	durationSet bool
	Timeout     time.Duration
}

// IsVideo reports whether this config targets Seedance 2.5 text-to-video.
func (c Config) IsVideo() bool {
	return kindOf(c.Model) == kindSeedanceVideo
}

// FromEnv reads Action inputs from INPUT_* environment variables.
func FromEnv() (Config, error) {
	cfg := Config{
		APIKey:      actionInput("api-key"),
		Prompt:      actionInput("prompt"),
		Output:      actionInput("output"),
		Model:       actionInput("model"),
		AspectRatio: actionInput("aspect-ratio"),
		Resolution:  actionInput("resolution"),
		Timeout:     10 * time.Minute,
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.Resolution == "" {
		cfg.Resolution = "720p"
	}

	rawDuration := actionInput("duration")
	if rawDuration != "" {
		n, err := strconv.Atoi(rawDuration)
		if err != nil {
			return Config{}, fmt.Errorf("invalid duration %q: %w", rawDuration, err)
		}
		cfg.Duration = n
		cfg.durationSet = true
	}

	if cfg.IsVideo() {
		if cfg.AspectRatio == "" {
			cfg.AspectRatio = "16:9"
		}
		if !cfg.durationSet {
			cfg.Duration = 5
		}
	} else if cfg.AspectRatio == "" {
		cfg.AspectRatio = "4:3"
	}

	if raw := actionInput("timeout"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid timeout %q: %w", raw, err)
		}
		if d <= 0 {
			return Config{}, fmt.Errorf("timeout must be positive, got %q", raw)
		}
		cfg.Timeout = d
	}

	return cfg, cfg.Validate()
}

// Validate checks required fields and credential format.
func (c Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("api-key is required")
	}
	if err := validateAPIKey(c.APIKey); err != nil {
		return err
	}
	if c.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if c.Output == "" {
		return fmt.Errorf("output is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if kindOf(c.Model) == kindUnsupportedVideo {
		return fmt.Errorf("unsupported video model %q; only %s is supported", c.Model, VideoModel)
	}
	if c.IsVideo() {
		return validateVideo(c)
	}
	if c.durationSet || c.Duration != 0 {
		return fmt.Errorf("duration is only supported for %s", VideoModel)
	}
	switch c.Resolution {
	case "", "720p", "1080p":
	default:
		return fmt.Errorf("resolution must be 720p or 1080p, got %q", c.Resolution)
	}
	return nil
}

func validateVideo(c Config) error {
	if c.Duration < 4 || c.Duration > 30 {
		return fmt.Errorf("duration must be between 4 and 30 seconds, got %d", c.Duration)
	}
	switch c.AspectRatio {
	case "16:9", "4:3", "1:1", "3:4", "9:16", "21:9":
	default:
		return fmt.Errorf("aspect-ratio must be 16:9, 4:3, 1:1, 3:4, 9:16, or 21:9, got %q", c.AspectRatio)
	}
	switch c.Resolution {
	case "480p", "720p", "1080p":
	default:
		return fmt.Errorf("resolution must be 480p, 720p, or 1080p, got %q", c.Resolution)
	}
	return nil
}

type modelKind int

const (
	kindImage modelKind = iota
	kindSeedanceVideo
	kindUnsupportedVideo
)

// videoWorkflows are path endings for video endpoints this Action does not call.
// Seedance 2.5 text-to-video is matched by its full path before this list.
var videoWorkflows = map[string]struct{}{
	"text-to-video":  {},
	"image-to-video": {},
}

func kindOf(model string) modelKind {
	switch model {
	case VideoModel:
		return kindSeedanceVideo
	}
	if _, ok := videoWorkflows[workflowName(model)]; ok {
		return kindUnsupportedVideo
	}
	return kindImage
}

func workflowName(model string) string {
	if i := strings.LastIndex(model, "/"); i >= 0 {
		return model[i+1:]
	}
	return model
}

// actionInput reads an Action input from the environment.
//
// Docker container actions pass INPUT_<name> with hyphens kept
// (api-key → INPUT_API-KEY). Some docs and JS actions use underscores
// (INPUT_API_KEY). Accept both so local tests and hosted runners work.
func actionInput(name string) string {
	upper := strings.ToUpper(name)
	underscored := strings.ReplaceAll(upper, "-", "_")
	if v := strings.TrimSpace(os.Getenv("INPUT_" + underscored)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("INPUT_" + upper))
}

func validateAPIKey(key string) error {
	parts := strings.Split(key, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("api-key must be KEY_ID:KEY_SECRET")
	}
	return nil
}
