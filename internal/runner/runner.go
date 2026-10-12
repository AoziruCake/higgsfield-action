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

	job := mediaJobFor(cfg)
	submitted, status, err := generate(ctx, client, job)
	if err != nil {
		return err
	}

	mediaURL, err := job.completedURL(status)
	if err != nil {
		return err
	}

	if err := saveMedia(ctx, client, dest.abs, mediaURL, job.kind); err != nil {
		return err
	}

	return writeOutputs(outputs, submitted, status, job.outputKey, mediaURL, dest.rel)
}

// mediaJob is the image or video submit/read pair for one model.
type mediaJob struct {
	outputKey string
	kind      string
	submit    func(context.Context, *higgsfield.Client) (higgsfield.SubmitResponse, error)
	urlFrom   func(higgsfield.RequestStatus) (string, bool)
}

func mediaJobFor(cfg config.Config) mediaJob {
	if cfg.IsVideo() {
		return mediaJob{
			outputKey: "video-url",
			kind:      "video",
			submit: func(ctx context.Context, client *higgsfield.Client) (higgsfield.SubmitResponse, error) {
				return client.SubmitVideo(ctx, cfg.Model, higgsfield.VideoRequest{
					Prompt:      cfg.Prompt,
					Duration:    cfg.Duration,
					Resolution:  cfg.Resolution,
					AspectRatio: cfg.AspectRatio,
				})
			},
			urlFrom: func(status higgsfield.RequestStatus) (string, bool) {
				return status.VideoURL()
			},
		}
	}
	return mediaJob{
		outputKey: "image-url",
		kind:      "image",
		submit: func(ctx context.Context, client *higgsfield.Client) (higgsfield.SubmitResponse, error) {
			return client.SubmitImage(ctx, cfg.Model, higgsfield.ImageRequest{
				Prompt:      cfg.Prompt,
				AspectRatio: cfg.AspectRatio,
				Resolution:  cfg.Resolution,
			})
		},
		urlFrom: func(status higgsfield.RequestStatus) (string, bool) {
			return status.FirstImageURL()
		},
	}
}

func (j mediaJob) completedURL(status higgsfield.RequestStatus) (string, error) {
	url, ok := j.urlFrom(status)
	if !ok {
		return "", fmt.Errorf("generation completed but no %s URL in response (request_id=%s)", j.kind, status.RequestID)
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return outputDest{}, fmt.Errorf("create output directory: %w", err)
	}
	if err := probeWritableDir(dir); err != nil {
		return outputDest{}, fmt.Errorf("output path is not writable: %w", err)
	}
	return outputDest{abs: abs, rel: rel}, nil
}

func probeWritableDir(dir string) error {
	file, err := os.CreateTemp(dir, ".higgsfield-action-write-*")
	if err != nil {
		return err
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Remove(name)
}

func generate(ctx context.Context, client *higgsfield.Client, job mediaJob) (higgsfield.SubmitResponse, higgsfield.RequestStatus, error) {
	submitted, err := job.submit(ctx, client)
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

func saveMedia(ctx context.Context, client *higgsfield.Client, outputPath, mediaURL, kind string) error {
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	if err := client.Download(ctx, mediaURL, file); err != nil {
		return fmt.Errorf("download %s: %w", kind, err)
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
