package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AoziruCake/higgsfield-action/internal/config"
	"github.com/AoziruCake/higgsfield-action/internal/gha"
	"github.com/AoziruCake/higgsfield-action/internal/higgsfield"
)

func TestExecute_success(t *testing.T) {
	imageBody := []byte("png-bytes")
	srv := httptest.NewServer(nil)
	t.Cleanup(srv.Close)
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/higgsfield-ai/soul/v2/standard"):
			_ = json.NewEncoder(w).Encode(higgsfield.SubmitResponse{
				Status:    higgsfield.StatusQueued,
				RequestID: "req-1",
				StatusURL: srv.URL + "/requests/req-1/status",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/requests/req-1/status":
			_ = json.NewEncoder(w).Encode(higgsfield.RequestStatus{
				Status:    higgsfield.StatusCompleted,
				RequestID: "req-1",
				Images:    []higgsfield.MediaOutput{{URL: srv.URL + "/image.png"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/image.png":
			_, _ = w.Write(imageBody)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	root := t.TempDir()
	outputFile := filepath.Join(root, "out.png")
	outputsPath := filepath.Join(root, "github-output")
	t.Setenv("GITHUB_WORKSPACE", root)
	t.Setenv("GITHUB_OUTPUT", outputsPath)

	cfg := config.Config{
		APIKey:      "id:secret",
		Prompt:      "A cat",
		Output:      "out.png",
		Model:       config.DefaultModel,
		AspectRatio: "4:3",
		Resolution:  "720p",
		Timeout:     time.Minute,
	}

	client := higgsfield.NewClient(cfg.APIKey, srv.Client()).WithBaseURL(srv.URL)
	outputs, err := gha.NewOutputWriterFromEnv()
	if err != nil {
		t.Fatalf("NewOutputWriterFromEnv: %v", err)
	}

	if err := execute(context.Background(), cfg, client, outputs); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("ReadFile output: %v", err)
	}
	if string(got) != string(imageBody) {
		t.Fatalf("output file = %q", got)
	}

	raw, err := os.ReadFile(outputsPath)
	if err != nil {
		t.Fatalf("ReadFile outputs: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "request-id=req-1\n") {
		t.Fatalf("outputs = %q", content)
	}
	if !strings.Contains(content, "image-url="+srv.URL+"/image.png\n") {
		t.Fatalf("outputs = %q", content)
	}
	if !strings.Contains(content, "output-path=out.png\n") {
		t.Fatalf("outputs = %q", content)
	}
}

func TestExecute_preflightSkipsAPI(t *testing.T) {
	var posts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts++
		t.Errorf("unexpected API call: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "output"), []byte("not-a-directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_WORKSPACE", root)
	t.Setenv("GITHUB_OUTPUT", filepath.Join(root, "github-output"))

	cfg := config.Config{
		APIKey:      "id:secret",
		Prompt:      "A cat",
		Output:      "output/integration.png",
		Model:       config.DefaultModel,
		AspectRatio: "4:3",
		Resolution:  "720p",
		Timeout:     time.Minute,
	}

	client := higgsfield.NewClient(cfg.APIKey, srv.Client()).WithBaseURL(srv.URL)
	outputs, err := gha.NewOutputWriterFromEnv()
	if err != nil {
		t.Fatalf("NewOutputWriterFromEnv: %v", err)
	}

	err = execute(context.Background(), cfg, client, outputs)
	if err == nil {
		t.Fatal("expected preflight error")
	}
	if posts != 0 {
		t.Fatalf("API calls = %d, want 0", posts)
	}
}

func TestExecute_preflightDoesNotCreateOutputFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "not_enough_credits"})
	}))
	t.Cleanup(srv.Close)

	root := t.TempDir()
	outputFile := filepath.Join(root, "output", "integration.png")
	t.Setenv("GITHUB_WORKSPACE", root)
	t.Setenv("GITHUB_OUTPUT", filepath.Join(root, "github-output"))

	cfg := config.Config{
		APIKey:      "id:secret",
		Prompt:      "A cat",
		Output:      "output/integration.png",
		Model:       config.DefaultModel,
		AspectRatio: "4:3",
		Resolution:  "720p",
		Timeout:     time.Minute,
	}

	client := higgsfield.NewClient(cfg.APIKey, srv.Client()).WithBaseURL(srv.URL)
	outputs, err := gha.NewOutputWriterFromEnv()
	if err != nil {
		t.Fatalf("NewOutputWriterFromEnv: %v", err)
	}

	if err := execute(context.Background(), cfg, client, outputs); err == nil {
		t.Fatal("expected generate error")
	}
	if _, err := os.Stat(outputFile); !os.IsNotExist(err) {
		t.Fatalf("output file should not exist after failed generate: %v", err)
	}
}
