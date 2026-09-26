package higgsfield

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const maxDownloadBytes = 100 << 20 // 100 MiB; rejects a runaway CDN response

// Download fetches media from a completed request's CDN URL.
// The request is unauthenticated: output URLs are public for a limited time.
func (c *Client) Download(ctx context.Context, mediaURL string, w io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download media: %w", err)
	}
	defer resp.Body.Close()

	if !isSuccessStatus(resp.StatusCode) {
		return fmt.Errorf("download media: HTTP %d", resp.StatusCode)
	}

	// Read one extra byte so we can tell "exactly at the limit" from "over the limit".
	limited := io.LimitReader(resp.Body, maxDownloadBytes+1)
	n, err := io.Copy(w, limited)
	if err != nil {
		return fmt.Errorf("save media: %w", err)
	}
	if n > maxDownloadBytes {
		return fmt.Errorf("download media: exceeds maximum size (%d bytes)", maxDownloadBytes)
	}
	return nil
}
