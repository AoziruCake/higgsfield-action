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

// Duration is a video length in seconds.
// The zero Duration means the input was omitted. ExplicitDuration(0) means
// the caller set zero, which image models reject and video models reject
// as out of range.
type Duration struct {
	seconds int
	present bool
}

// ExplicitDuration records a duration the caller set, including zero.
func ExplicitDuration(seconds int) Duration {
	return Duration{seconds: seconds, present: true}
}

// Present reports whether the caller set a duration.
func (d Duration) Present() bool { return d.present }

// Seconds returns the length. It is zero when the duration was omitted.
func (d Duration) Seconds() int { return d.seconds }

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
	Duration    Duration // video length; zero means omitted
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
	if cfg.Resolution == "" {
		cfg.Resolution = "720p"
	}

	rawDuration := actionInput("duration")
	if rawDuration != "" {
		n, err := strconv.Atoi(rawDuration)
		if err != nil {
			return Config{}, fmt.Errorf("invalid duration %q: %w", rawDuration, err)
		}
		cfg.Duration = ExplicitDuration(n)
	}
	cfg.applyModelDefaults()

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
	profile, ok := classify(c.Model)
	if !ok {
		return unsupportedModel(c.Model)
	}
	return profile.validate(c)
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
