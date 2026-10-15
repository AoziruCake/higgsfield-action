// Package runner runs one generation Action: submit, poll, save, outputs.
package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AoziruCake/higgsfield-action/internal/config"
	"github.com/AoziruCake/higgsfield-action/internal/gha"
	"github.com/AoziruCake/higgsfield-action/internal/higgsfield"
	"github.com/AoziruCake/higgsfield-action/internal/workspace"
)

// Run executes one Action invocation: submit, poll, download, write outputs.
func Run(ctx context.Context, cfg config.Config) error {
	outputs, err := gha.NewOutputWriterFromEnv()
	if err != nil {
		return err
	}
	// nil selects NewClient's 60s timeout for status calls. Download uses the
	// Action context instead of that timeout.
	return execute(ctx, cfg, higgsfield.NewClient(cfg.APIKey, nil), outputs)
}

func execute(ctx context.Context, cfg config.Config, client *higgsfield.Client, outputs *gha.OutputWriter) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Fail before submit: a later mkdir/output failure would still consume credits.
	dest, err := prepareOutput(cfg.Output)
	if err != nil {
		return err
	}
	if err := outputs.EnsureWritable(); err != nil {
		return err
	}

	job, err := mediaJobFor(cfg)
	if err != nil {
		return err
	}
	submitted, status, err := generate(ctx, client, cfg, job)
	if err != nil {
		return err
	}

	mediaURL, err := job.completedURL(status)
	if err != nil {
		return err
	}

	if err := saveMedia(ctx, client, dest.abs, mediaURL, job.noun); err != nil {
		return err
	}

	return writeOutputs(outputs, submitted, status, job.outputKey, mediaURL, dest.rel)
}

// mediaJob is the submit/read pair for one supported model.
// Output key and noun come from config.Media, so this package does not
// restate which model is an image or a video.
type mediaJob struct {
	outputKey string
	noun      string
	submit    func(context.Context, *higgsfield.Client, config.Config) (higgsfield.SubmitResponse, error)
	urlFrom   func(higgsfield.RequestStatus) (string, bool)
}

type jobStyle struct {
	submit  func(context.Context, *higgsfield.Client, config.Config) (higgsfield.SubmitResponse, error)
	urlFrom func(higgsfield.RequestStatus) (string, bool)
}

// jobStyles is the only place that picks a request body and a result URL.
// Keys are config.Kind, the same distinction config uses for defaults and validation.
var jobStyles = map[config.Kind]jobStyle{
	config.KindImage: {
		submit:  submitImage,
		urlFrom: higgsfield.RequestStatus.FirstImageURL,
	},
	config.KindVideo: {
		submit:  submitVideo,
		urlFrom: higgsfield.RequestStatus.VideoURL,
	},
}

func mediaJobFor(cfg config.Config) (mediaJob, error) {
	media, err := cfg.Media()
	if err != nil {
		return mediaJob{}, err
	}
	style, ok := jobStyles[media.Kind]
	if !ok {
		return mediaJob{}, fmt.Errorf("unsupported model %q", cfg.Model)
	}
	return mediaJob{
		outputKey: media.OutputKey,
		noun:      media.Noun,
		submit:    style.submit,
		urlFrom:   style.urlFrom,
	}, nil
}

func submitImage(ctx context.Context, client *higgsfield.Client, cfg config.Config) (higgsfield.SubmitResponse, error) {
	return client.SubmitImage(ctx, cfg.Model, higgsfield.ImageRequest{PromptFields: promptFields(cfg)})
}

func submitVideo(ctx context.Context, client *higgsfield.Client, cfg config.Config) (higgsfield.SubmitResponse, error) {
	return client.SubmitVideo(ctx, cfg.Model, higgsfield.VideoRequest{
		PromptFields: promptFields(cfg),
		Duration:     cfg.Duration.Seconds(),
	})
}

func promptFields(cfg config.Config) higgsfield.PromptFields {
	return higgsfield.PromptFields{
		Prompt:      cfg.Prompt,
		AspectRatio: cfg.AspectRatio,
		Resolution:  cfg.Resolution,
	}
}

func (j mediaJob) completedURL(status higgsfield.RequestStatus) (string, error) {
	url, ok := j.urlFrom(status)
	if !ok {
		return "", fmt.Errorf("generation completed but no %s URL in response (request_id=%s)", j.noun, status.RequestID)
	}
	return url, nil
}

type outputDest struct {
	abs string
	rel string
}

func prepareOutput(output string) (outputDest, error) {
	abs, err := workspace.ResolveOutputPath(output)
	if err != nil {
		return outputDest{}, fmt.Errorf("resolve output path: %w", err)
	}
	rel, err := workspace.Rel(abs)
	if err != nil {
		return outputDest{}, fmt.Errorf("resolve output path: %w", err)
	}
	dir := filepath.Dir(abs)
	if err := workspace.MkdirWithin(dir); err != nil {
		return outputDest{}, fmt.Errorf("create output directory: %w", err)
	}
	if err := workspace.ProbeWritable(dir); err != nil {
		return outputDest{}, fmt.Errorf("output path is not writable: %w", err)
	}
	return outputDest{abs: abs, rel: rel}, nil
}

func generate(ctx context.Context, client *higgsfield.Client, cfg config.Config, job mediaJob) (higgsfield.SubmitResponse, higgsfield.RequestStatus, error) {
	submitted, err := job.submit(ctx, client, cfg)
	if err != nil {
		return submitted, higgsfield.RequestStatus{}, err
	}

	// Use the status_url from the submit response; do not build it from request_id.
	status, err := client.WaitForCompletion(ctx, submitted.StatusURL, higgsfield.DefaultPollOptions())
	if err != nil {
		return submitted, status, fmt.Errorf("wait for completion: %w", err)
	}
	return submitted, status, nil
}

func saveMedia(ctx context.Context, client *higgsfield.Client, outputPath, mediaURL, noun string) error {
	file, err := createOutput(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	if err := client.Download(ctx, mediaURL, file); err != nil {
		_ = file.Close()
		_ = os.Remove(outputPath)
		return fmt.Errorf("download %s: %w", noun, err)
	}
	return nil
}

func writeOutputs(outputs *gha.OutputWriter, submitted higgsfield.SubmitResponse, status higgsfield.RequestStatus, outputKey, mediaURL, outputPath string) error {
	requestID := status.RequestID
	if requestID == "" {
		requestID = submitted.RequestID
	}

	if err := outputs.Set("request-id", requestID); err != nil {
		return fmt.Errorf("set output request-id: %w", err)
	}
	if err := outputs.Set(outputKey, mediaURL); err != nil {
		return fmt.Errorf("set output %s: %w", outputKey, err)
	}
	if err := outputs.Set("output-path", outputPath); err != nil {
		return fmt.Errorf("set output output-path: %w", err)
	}
	return nil
}
