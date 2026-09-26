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
// Docker Actions set INPUT_API-KEY (hyphen kept). Some runners and local
// tests use INPUT_API_KEY. FromEnv accepts both.
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
	if cfg.AspectRatio == "" {
		cfg.AspectRatio = "4:3"
	}
	if cfg.Resolution == "" {
		cfg.Resolution = "720p"
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
	return nil
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
