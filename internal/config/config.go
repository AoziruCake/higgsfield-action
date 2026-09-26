// Package config reads GitHub Action inputs from INPUT_* environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// DefaultModel is Soul v2 Standard, the MVP image endpoint.
const DefaultModel = "higgsfield-ai/soul/v2/standard"

// Config holds GitHub Action inputs after parsing and validation.
//
// Container Actions expose each input as INPUT_<NAME> with hyphens turned into
// underscores (api-key → INPUT_API_KEY).
type Config struct {
	APIKey      string // KEY_ID:KEY_SECRET
	Prompt      string
	Output      string // workspace-relative path
	Model       string
	AspectRatio string
	Resolution  string
	Timeout     time.Duration
}

// FromEnv reads Action inputs from INPUT_* environment variables.
func FromEnv() (Config, error) {
	cfg := Config{
		APIKey:      strings.TrimSpace(os.Getenv("INPUT_API_KEY")),
		Prompt:      strings.TrimSpace(os.Getenv("INPUT_PROMPT")),
		Output:      strings.TrimSpace(os.Getenv("INPUT_OUTPUT")),
		Model:       strings.TrimSpace(os.Getenv("INPUT_MODEL")),
		AspectRatio: strings.TrimSpace(os.Getenv("INPUT_ASPECT_RATIO")),
		Resolution:  strings.TrimSpace(os.Getenv("INPUT_RESOLUTION")),
		Timeout:     10 * time.Minute,
	}

	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.AspectRatio == "" {
		cfg.AspectRatio = "4:3"
	}
	if cfg.Resolution == "" {
		cfg.Resolution = "720p"
	}

	if raw := strings.TrimSpace(os.Getenv("INPUT_TIMEOUT")); raw != "" {
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
	return nil
}

func validateAPIKey(key string) error {
	parts := strings.Split(key, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("api-key must be KEY_ID:KEY_SECRET")
	}
	return nil
}
