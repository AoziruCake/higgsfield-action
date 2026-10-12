package higgsfield_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AoziruCake/higgsfield-action/internal/higgsfield"
)

func TestDownload_success(t *testing.T) {
	t.Parallel()

	body := []byte("fake-image-bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/image.png" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	client := higgsfield.NewClient(testAPIKey, srv.Client())
	var buf bytes.Buffer
	if err := client.Download(context.Background(), srv.URL+"/image.png", &buf); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), body) {
		t.Fatalf("body = %q", buf.Bytes())
	}
}

func TestDownload_ignoresClientTimeout(t *testing.T) {
	t.Parallel()

	body := []byte("fake-video-bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	hc := srv.Client()
	hc.Timeout = time.Nanosecond
	client := higgsfield.NewClient(testAPIKey, hc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var buf bytes.Buffer
	if err := client.Download(ctx, srv.URL+"/out.mp4", &buf); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), body) {
		t.Fatalf("body = %q", buf.Bytes())
	}
}

func TestDownload_httpError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	client := higgsfield.NewClient(testAPIKey, srv.Client())
	err := client.Download(context.Background(), srv.URL+"/missing", ioDiscard{})
	if err == nil {
		t.Fatal("expected error")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
