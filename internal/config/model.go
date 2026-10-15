package config

import (
	"fmt"
)

// Kind identifies a supported generation endpoint.
type Kind int

const (
	// KindImage is a Soul-style text-to-image endpoint.
	KindImage Kind = iota
	// KindVideo is Seedance 2.5 text-to-video.
	KindVideo
)

// Media is the Action-facing description of a supported model.
// OutputKey is the GitHub output name. Noun is the word used in errors
// ("image" or "video").
type Media struct {
	Kind      Kind
	OutputKey string
	Noun      string
}

// Media returns the description for this config's model.
// Unsupported models return an error instead of an image description.
func (c Config) Media() (Media, error) {
	profile, ok := classify(c.Model)
	if !ok {
		return Media{}, unsupportedModel(c.Model)
	}
	return profile.media, nil
}

type durationRule struct {
	applies  bool
	min      int
	max      int
	fallback int
}

// modelProfile is the single description of a supported model: defaults,
// validation, and the Action output it writes.
type modelProfile struct {
	media           Media
	defaultAspect   string
	resolutions     map[string]struct{}
	resolutionError string
	aspects         map[string]struct{}
	aspectError     string
	duration        durationRule
}

var imageProfile = modelProfile{
	media:         Media{Kind: KindImage, OutputKey: "image-url", Noun: "image"},
	defaultAspect: "4:3",
	// Empty stays valid when Config is built directly. FromEnv fills 720p.
	resolutions:     setOf("", "720p", "1080p"),
	resolutionError: "resolution must be 720p or 1080p, got %q",
}

var videoProfile = modelProfile{
	media:           Media{Kind: KindVideo, OutputKey: "video-url", Noun: "video"},
	defaultAspect:   "16:9",
	resolutions:     setOf("480p", "720p", "1080p"),
	resolutionError: "resolution must be 480p, 720p, or 1080p, got %q",
	aspects:         setOf("16:9", "4:3", "1:1", "3:4", "9:16", "21:9"),
	aspectError:     "aspect-ratio must be 16:9, 4:3, 1:1, 3:4, 9:16, or 21:9, got %q",
	duration:        durationRule{applies: true, min: 4, max: 30, fallback: 5},
}

// classify returns the profile for a supported model.
// The boolean is false for every path this Action does not call.
func classify(model string) (modelProfile, bool) {
	switch model {
	case DefaultModel:
		return imageProfile, true
	case VideoModel:
		return videoProfile, true
	default:
		return modelProfile{}, false
	}
}

func (c *Config) applyModelDefaults() {
	profile, ok := classify(c.Model)
	if !ok {
		return
	}
	if c.AspectRatio == "" {
		c.AspectRatio = profile.defaultAspect
	}
	if profile.duration.applies && !c.Duration.Present() {
		c.Duration = ExplicitDuration(profile.duration.fallback)
	}
}

func (p modelProfile) validate(c Config) error {
	if !p.duration.applies {
		if c.Duration.Present() {
			return fmt.Errorf("duration is only supported for %s", VideoModel)
		}
	} else if n := c.Duration.Seconds(); n < p.duration.min || n > p.duration.max {
		return fmt.Errorf("duration must be between %d and %d seconds, got %d", p.duration.min, p.duration.max, n)
	}
	if len(p.aspects) > 0 {
		if _, ok := p.aspects[c.AspectRatio]; !ok {
			return fmt.Errorf(p.aspectError, c.AspectRatio)
		}
	}
	if _, ok := p.resolutions[c.Resolution]; !ok {
		return fmt.Errorf(p.resolutionError, c.Resolution)
	}
	return nil
}

func unsupportedModel(model string) error {
	return fmt.Errorf("unsupported model %q; supported models are %s and %s", model, DefaultModel, VideoModel)
}

func setOf(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
