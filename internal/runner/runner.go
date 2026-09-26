// Package runner runs one image-generation Action: submit, poll, save, outputs.
package runner

import (
	"context"
	"fmt"
	"net/http"
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
	return execute(ctx, cfg, higgsfield.NewClient(cfg.APIKey, &http.Client{}), outputs)
}

func execute(ctx context.Context, cfg config.Config, client *higgsfield.Client, outputs *gha.OutputWriter) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	submitted, status, err := generate(ctx, cfg, client)
	if err != nil {
		return err
	}

	imageURL, ok := status.FirstImageURL()
	if !ok {
		return fmt.Errorf("generation completed but no image URL in response (request_id=%s)", status.RequestID)
	}

	outputPath, err := saveImage(ctx, client, cfg.Output, imageURL)
	if err != nil {
		return err
	}

	return writeOutputs(outputs, submitted, status, imageURL, outputPath)
}

func generate(ctx context.Context, cfg config.Config, client *higgsfield.Client) (higgsfield.SubmitResponse, higgsfield.RequestStatus, error) {
	submitted, err := client.SubmitImage(ctx, cfg.Model, higgsfield.ImageRequest{
		Prompt:      cfg.Prompt,
		AspectRatio: cfg.AspectRatio,
		Resolution:  cfg.Resolution,
	})
	if err != nil {
		return submitted, higgsfield.RequestStatus{}, fmt.Errorf("submit image: %w", err)
	}

	// Use the status_url from the submit response; do not build it from request_id.
	status, err := client.WaitForCompletion(ctx, submitted.StatusURL, higgsfield.DefaultPollOptions())
	if err != nil {
		return submitted, status, fmt.Errorf("wait for completion: %w", err)
	}
	return submitted, status, nil
}

func saveImage(ctx context.Context, client *higgsfield.Client, output, imageURL string) (string, error) {
	outputPath, err := workspace.ResolveOutputPath(output)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	if err := client.Download(ctx, imageURL, file); err != nil {
		return "", fmt.Errorf("download image: %w", err)
	}
	return outputPath, nil
}

func writeOutputs(outputs *gha.OutputWriter, submitted higgsfield.SubmitResponse, status higgsfield.RequestStatus, imageURL, outputPath string) error {
	requestID := status.RequestID
	if requestID == "" {
		requestID = submitted.RequestID
	}

	if err := outputs.Set("request-id", requestID); err != nil {
		return fmt.Errorf("set output request-id: %w", err)
	}
	if err := outputs.Set("image-url", imageURL); err != nil {
		return fmt.Errorf("set output image-url: %w", err)
	}
	if err := outputs.Set("output-path", outputPath); err != nil {
		return fmt.Errorf("set output output-path: %w", err)
	}
	return nil
}
