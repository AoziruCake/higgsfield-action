package higgsfield

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const maxDownloadBytes = 512 << 20 // 512 MiB; 1080p 30s high-bitrate video can exceed 100 MiB

// Download fetches media from a completed request's CDN URL.
// The request is unauthenticated: output URLs are public for a limited time.
//
// The Action timeout on ctx is the deadline. The JSON client's Timeout is not
// applied, so a large video can finish inside the Action timeout.
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

	resp, err := c.downloadHTTP().Do(req)
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

// downloadHTTP shares the API client's transport but drops its Timeout so the
// caller's context is the only deadline.
func (c *Client) downloadHTTP() *http.Client {
	return &http.Client{
		Transport:     c.httpClient.Transport,
		CheckRedirect: c.httpClient.CheckRedirect,
		Jar:           c.httpClient.Jar,
	}
}
