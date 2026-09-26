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

// Run executes the full image generation workflow for the GitHub Action.
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

	submitted, err := client.SubmitImage(ctx, cfg.Model, higgsfield.ImageRequest{
		Prompt:      cfg.Prompt,
		AspectRatio: cfg.AspectRatio,
		Resolution:  cfg.Resolution,
	})
	if err != nil {
		return fmt.Errorf("submit image: %w", err)
	}

	status, err := client.WaitForCompletion(ctx, submitted.StatusURL, higgsfield.DefaultPollOptions())
	if err != nil {
		return fmt.Errorf("wait for completion: %w", err)
	}

	imageURL, ok := status.FirstImageURL()
	if !ok {
		return fmt.Errorf("generation completed but no image URL in response (request_id=%s)", status.RequestID)
	}

	outputPath, err := workspace.ResolveOutputPath(cfg.Output)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	if err := client.Download(ctx, imageURL, file); err != nil {
		return fmt.Errorf("download image: %w", err)
	}

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
