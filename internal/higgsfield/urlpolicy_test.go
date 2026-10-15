package higgsfield_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/AoziruCake/higgsfield-action/internal/higgsfield"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetStatus_rejectsForeignHost(t *testing.T) {
	t.Parallel()

	client := higgsfield.NewClient(testAPIKey, &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("status request must not be sent")
			return nil, nil
		}),
	})
	_, err := client.GetStatus(context.Background(), "https://evil.example/requests/x/status")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDownload_allowsCDNWithoutCredentials(t *testing.T) {
	t.Parallel()

	for _, mediaURL := range []string{
		"https://cdn.higgsfield.ai/a.png",
		"https://d8j0ntlcm91z4.cloudfront.net/a.mp4",
	} {
		t.Run(mediaURL, func(t *testing.T) {
			client := higgsfield.NewClient(testAPIKey, &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if r.Header.Get("Authorization") != "" {
						t.Fatal("download sent credentials")
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader("bytes")),
						Request:    r,
					}, nil
				}),
			})
			var buf bytes.Buffer
			if err := client.Download(context.Background(), mediaURL, &buf); err != nil {
				t.Fatal(err)
			}
			if buf.String() != "bytes" {
				t.Fatalf("body = %q", buf.String())
			}
		})
	}
}

func TestDownload_rejectsUntrustedURL(t *testing.T) {
	t.Parallel()

	client := higgsfield.NewClient(testAPIKey, &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("download must not be sent")
			return nil, nil
		}),
	})
	for _, mediaURL := range []string{
		"http://169.254.169.254/latest/meta-data/",
		"https://evil.example/a.png",
		"https://user:pass@cdn.higgsfield.ai/a.png",
		"http://cdn.higgsfield.ai/a.png",
		"file:///etc/passwd",
	} {
		t.Run(mediaURL, func(t *testing.T) {
			err := client.Download(context.Background(), mediaURL, io.Discard)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDownload_rejectsRedirectOffHost(t *testing.T) {
	t.Parallel()

	var calls int
	client := higgsfield.NewClient(testAPIKey, &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{
				StatusCode: http.StatusFound,
				Header:     http.Header{"Location": []string{"https://169.254.169.254/latest/meta-data/"}},
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    r,
			}, nil
		}),
	})
	var buf bytes.Buffer
	err := client.Download(context.Background(), "https://cdn.higgsfield.ai/a.png", &buf)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("requests = %d, want 1", calls)
	}
	if buf.Len() != 0 {
		t.Fatalf("wrote %q", buf.String())
	}
}
