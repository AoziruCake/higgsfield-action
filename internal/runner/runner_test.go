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
	const imageBody = "png-bytes"
	fileBody, content, baseURL := executeSuccess(t, config.Config{
		APIKey:      "id:secret",
		Prompt:      "A cat",
		Output:      "out.png",
		Model:       config.DefaultModel,
		AspectRatio: "4:3",
		Resolution:  "720p",
		Timeout:     time.Minute,
	}, func(baseURL string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/higgsfield-ai/soul/v2/standard"):
				_ = json.NewEncoder(w).Encode(higgsfield.SubmitResponse{
					Status:    higgsfield.StatusQueued,
					RequestID: "req-1",
					StatusURL: baseURL + "/requests/req-1/status",
				})
			case r.Method == http.MethodGet && r.URL.Path == "/requests/req-1/status":
				_ = json.NewEncoder(w).Encode(higgsfield.RequestStatus{
					Status:    higgsfield.StatusCompleted,
					RequestID: "req-1",
					Images:    []higgsfield.MediaOutput{{URL: baseURL + "/image.png"}},
				})
			case r.Method == http.MethodGet && r.URL.Path == "/image.png":
				_, _ = w.Write([]byte(imageBody))
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})
	})

	if fileBody != imageBody {
		t.Fatalf("output file = %q", fileBody)
	}
	if !strings.Contains(content, "request-id=req-1\n") {
		t.Fatalf("outputs = %q", content)
	}
	if !strings.Contains(content, "image-url="+baseURL+"/image.png\n") {
		t.Fatalf("outputs = %q", content)
	}
	if !strings.Contains(content, "output-path=out.png\n") {
		t.Fatalf("outputs = %q", content)
	}
	if strings.Contains(content, "video-url=") {
		t.Fatalf("image run set video-url: %q", content)
	}
}

func TestExecute_videoSuccess(t *testing.T) {
	const videoBody = "mp4-bytes"
	fileBody, content, baseURL := executeSuccess(t, config.Config{
		APIKey:      "id:secret",
		Prompt:      "A cat",
		Output:      "out.mp4",
		Model:       config.VideoModel,
		AspectRatio: "16:9",
		Resolution:  "720p",
		Duration:    config.ExplicitDuration(5),
		Timeout:     time.Minute,
	}, func(baseURL string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/bytedance/seedance-2.5/text-to-video"):
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode body: %v", err)
				}
				if len(body) != 4 {
					t.Errorf("body fields = %d, want 4", len(body))
				}
				if string(body["duration"]) != "5" {
					t.Errorf("duration = %s", body["duration"])
				}
				_ = json.NewEncoder(w).Encode(higgsfield.SubmitResponse{
					Status:    higgsfield.StatusQueued,
					RequestID: "req-v",
					StatusURL: baseURL + "/requests/req-v/status",
				})
			case r.Method == http.MethodGet && r.URL.Path == "/requests/req-v/status":
				_ = json.NewEncoder(w).Encode(higgsfield.RequestStatus{
					Status:    higgsfield.StatusCompleted,
					RequestID: "req-v",
					Video:     &higgsfield.MediaOutput{URL: baseURL + "/out.mp4"},
				})
			case r.Method == http.MethodGet && r.URL.Path == "/out.mp4":
				w.Header().Set("Content-Type", "video/mp4")
				_, _ = w.Write([]byte(videoBody))
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})
	})

	if fileBody != videoBody {
		t.Fatalf("output file = %q", fileBody)
	}
	if !strings.Contains(content, "request-id=req-v\n") {
		t.Fatalf("outputs = %q", content)
	}
	if !strings.Contains(content, "video-url="+baseURL+"/out.mp4\n") {
		t.Fatalf("outputs = %q", content)
	}
	if strings.Contains(content, "image-url=") {
		t.Fatalf("video run set image-url: %q", content)
	}
	if !strings.Contains(content, "output-path=out.mp4\n") {
		t.Fatalf("outputs = %q", content)
	}
}

func executeSuccess(t *testing.T, cfg config.Config, handler func(baseURL string) http.Handler) (fileBody, outputs, baseURL string) {
	t.Helper()

	srv := httptest.NewServer(nil)
	t.Cleanup(srv.Close)
	srv.Config.Handler = handler(srv.URL)

	root := t.TempDir()
	outputFile := filepath.Join(root, filepath.FromSlash(cfg.Output))
	outputsPath := filepath.Join(root, "github-output")
	t.Setenv("GITHUB_WORKSPACE", root)
	t.Setenv("GITHUB_OUTPUT", outputsPath)

	client := higgsfield.NewClient(cfg.APIKey, srv.Client()).WithBaseURL(srv.URL)
	writer, err := gha.NewOutputWriterFromEnv()
	if err != nil {
		t.Fatalf("NewOutputWriterFromEnv: %v", err)
	}
	if err := execute(context.Background(), cfg, client, writer); err != nil {
		t.Fatalf("execute: %v", err)
	}

	rawFile, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("ReadFile output: %v", err)
	}
	rawOutputs, err := os.ReadFile(outputsPath)
	if err != nil {
		t.Fatalf("ReadFile outputs: %v", err)
	}
	return string(rawFile), string(rawOutputs), srv.URL
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
